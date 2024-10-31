package stages

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
	rollouts "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/kargo"
	"github.com/akuity/kargo/internal/kubeclient"
	libEvent "github.com/akuity/kargo/internal/kubernetes/event"
	"github.com/akuity/kargo/internal/logging"
)

type ControlFlowStageReconciler struct {
	cfg           ReconcilerConfig
	client        client.Client
	eventRecorder record.EventRecorder
}

// NewControlFlowStageReconciler returns a new control flow Stage reconciler.
// After creating the reconciler, call SetupWithManager to register it with a
// controller manager.
func NewControlFlowStageReconciler(
	cfg ReconcilerConfig,
) *ControlFlowStageReconciler {
	return &ControlFlowStageReconciler{
		cfg: cfg,
	}
}

// SetupWithManager sets up the control flow Stage reconciler with the given
// controller manager. It registers the reconciler with the manager and sets up
// watches on the required objects.
func (r *ControlFlowStageReconciler) SetupWithManager(
	ctx context.Context,
	mgr ctrl.Manager,
	sharedIndexer client.FieldIndexer,
) error {
	// Configure client and event recorder using manager.
	r.client = mgr.GetClient()
	r.eventRecorder = libEvent.NewRecorder(ctx, mgr.GetScheme(), mgr.GetClient(), r.cfg.Name())

	// This index is used to find all Freight that are directly available from
	// a Warehouse. It is used to find Freight that can be sourced directly from
	// the Warehouse for the control flow Stage.
	if err := sharedIndexer.IndexField(
		ctx,
		&kargoapi.Freight{},
		indexer.FreightByWarehouseField,
		indexer.FreightByWarehouse,
	); err != nil {
		return fmt.Errorf("error setting up index for Freight by Warehouse: %w", err)
	}

	// This index is used to find and watch all Freight that have been verified
	// in a specific Stage (upstream) to which the control flow Stage is the
	// downstream consumer.
	if err := sharedIndexer.IndexField(
		ctx,
		&kargoapi.Freight{},
		indexer.FreightByVerifiedStagesField,
		indexer.FreightByVerifiedStages,
	); err != nil {
		return fmt.Errorf("error setting up index for Freight by verified Stages: %w", err)
	}

	// This index is solely used to garbage collect any Freight that was
	// to a Stage before it became a control flow Stage. It is not used for
	// the actual reconciliation process beyond facilitating the garbage
	// collection of related objects when the Stage is deleted.
	if err := sharedIndexer.IndexField(
		ctx,
		&kargoapi.Freight{},
		indexer.FreightApprovedForStagesField,
		indexer.FreightApprovedForStages,
	); err != nil {
		return fmt.Errorf("error setting up index for Freight approved for Stages: %w", err)
	}

	// This index is used by a watch on Stages to find all Stages that have a
	// specific Stage as an upstream Stage.
	if err := sharedIndexer.IndexField(
		ctx,
		&kargoapi.Stage{},
		indexer.StagesByUpstreamStagesField,
		indexer.StagesByUpstreamStages,
	); err != nil {
		return fmt.Errorf("error setting up index for Stages by upstream Stages: %w", err)
	}

	// This index is used by a watch on Stages to find all Stages that have a
	// specific Warehouse as an upstream Warehouse.
	if err := sharedIndexer.IndexField(
		ctx,
		&kargoapi.Stage{},
		indexer.StagesByWarehouseField,
		indexer.StagesByWarehouse,
	); err != nil {
		return fmt.Errorf("error setting up index for Stages by Warehouse: %w", err)
	}

	// Build the controller with the reconciler.
	c, err := ctrl.NewControllerManagedBy(mgr).
		For(&kargoapi.Stage{}).
		Named("control_flow_stage").
		WithOptions(controller.CommonOptions(r.cfg.MaxConcurrentControlFlowReconciles)).
		WithEventFilter(
			predicate.And(
				IsControlFlowStage(true),
				predicate.Or(
					predicate.GenerationChangedPredicate{},
					kargo.RefreshRequested{},
				),
			),
		).
		Build(r)
	if err != nil {
		return fmt.Errorf("error building control flow Stage reconciler: %w", err)
	}

	// Configure the watches.
	// Changes to these objects that match the constraints from the predicates
	// will enqueue a reconciliation for the related Stage(s).

	// Watch for Freight that are directly available from a Warehouse.
	if err := c.Watch(
		source.Kind(
			mgr.GetCache(),
			&kargoapi.Freight{},
			&downstreamStageEnqueuer[*kargoapi.Freight]{
				kargoClient: mgr.GetClient(),
			},
		),
	); err != nil {
		return fmt.Errorf("unable to watch Freight from upstream Stages: %w", err)
	}

	// Watch for Freight that have been verified in upstream Stages.
	if err := c.Watch(
		source.Kind(
			mgr.GetCache(),
			&kargoapi.Freight{},
			&downstreamStageEnqueuer[*kargoapi.Freight]{
				kargoClient:          mgr.GetClient(),
				forControlFlowStages: true,
			},
		),
	); err != nil {
		return fmt.Errorf("unable to watch Freight verified in upstream Stages: %w", err)
	}

	logging.LoggerFromContext(ctx).Info(
		"Initialized control flow Stage reconciler",
		"maxConcurrentReconciles", r.cfg.MaxConcurrentControlFlowReconciles,
	)

	return nil
}

// Reconcile reconciles the given control flow Stage.
func (r *ControlFlowStageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"namespace", req.NamespacedName.Namespace,
		"stage", req.NamespacedName.Name,
		"controlFlow", true,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	// Find the Stage.
	stage := &kargoapi.Stage{}
	if err := r.client.Get(ctx, req.NamespacedName, stage); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Safety check: do not reconcile Stages that are not control flow Stages.
	if !stage.IsControlFlow() {
		return ctrl.Result{}, nil
	}

	// Handle deletion of the Stage.
	if !stage.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, r.handleDelete(ctx, stage)
	}

	// Ensure the Stage has a finalizer and requeue if it was added.
	// The reason to requeue is to ensure that a possible deletion of the Stage
	// directly after the finalizer was added is handled without delay.
	if ok, err := kargoapi.EnsureFinalizer(ctx, r.client, stage); ok || err != nil {
		return ctrl.Result{Requeue: ok}, err
	}

	// Reconcile the Stage.
	logger.Debug("reconciling Stage")
	newStatus, reconcileErr := r.reconcile(ctx, stage, time.Now())
	logger.Debug("done reconciling Stage")

	// Patch the status of the Stage.
	if err := kubeclient.PatchStatus(ctx, r.client, stage, func(status *kargoapi.StageStatus) {
		*status = newStatus
	}); err != nil {
		// Prioritize the reconcile error if it exists.
		if reconcileErr != nil {
			logger.Error(err, "failed to update Stage status after reconciliation error")
			return ctrl.Result{}, reconcileErr
		}
		return ctrl.Result{}, fmt.Errorf("failed to update Stage status: %w", err)
	}
	return ctrl.Result{}, reconcileErr
}

// reconcile reconciles the given control flow Stage. It verifies the (newly)
// available Freight for the Stage, recording the verification results in the
// Freight objects and emitting events for the successful verifications.
//
// It returns the updated status of the Stage. The caller is responsible for
// updating the Stage with the returned status.
//
// In case of an error, the Stage status is updated with the error message.
func (r *ControlFlowStageReconciler) reconcile(
	ctx context.Context,
	stage *kargoapi.Stage,
	startTime time.Time,
) (kargoapi.StageStatus, error) {
	logger := logging.LoggerFromContext(ctx)

	// Always initialize the status of the Stage.
	newStatus := r.initializeStatus(stage)

	// Get the available Freight for the Stage.
	logger.Debug("getting available Freight")
	freight, err := r.getAvailableFreight(
		ctx,
		types.NamespacedName{
			Namespace: stage.Namespace,
			Name:      stage.Name,
		},
		stage.Spec.RequestedFreight,
	)
	if err != nil {
		newStatus.Message = err.Error()
		return newStatus, err
	}

	// If there is new Freight to verify, do so.
	if len(freight) > 0 {
		logger.Debug("found new Freight", "count", len(freight))

		logger.Debug("verifying Freight")
		if err = r.verifyFreight(ctx, stage, freight, startTime, time.Now()); err != nil {
			newStatus.Message = err.Error()
			return newStatus, err
		}
		logger.Debug("verified Freight", "count", len(freight))
	}

	return newStatus, nil
}

// initializeStatus initializes the status of the given Stage with the values
// that are common to all control flow Stages. It resets the status to a clean
// state, recording the current refresh token as having been handled.
func (r *ControlFlowStageReconciler) initializeStatus(stage *kargoapi.Stage) kargoapi.StageStatus {
	newStatus := stage.Status.DeepCopy()

	// Update the status with the new observed generation and phase.
	newStatus.Phase = kargoapi.StagePhaseNotApplicable
	if stage.Generation > stage.Status.ObservedGeneration {
		newStatus.ObservedGeneration = stage.Generation
	}

	// Reset any previous error message.
	newStatus.Message = ""

	// Record the current refresh token as having been handled.
	if token, ok := kargoapi.RefreshAnnotationValue(stage.GetAnnotations()); ok {
		newStatus.LastHandledRefresh = token
	}

	// Clear all the fields that are not relevant to this Stage type.
	newStatus.FreightHistory = nil
	newStatus.Health = nil
	newStatus.CurrentPromotion = nil
	newStatus.LastPromotion = nil
	newStatus.FreightSummary = "N/A"

	return *newStatus
}

// getAvailableFreight returns the list of available Freight for the given
// Stage. Freight is considered available if it can be sourced directly from
// the Warehouse or if it has been verified in upstream Stages. It excludes
// Freight that has already been verified in the given Stage.
func (r *ControlFlowStageReconciler) getAvailableFreight(
	ctx context.Context,
	stage types.NamespacedName,
	requested []kargoapi.FreightRequest,
) ([]kargoapi.Freight, error) {
	var availableFreight []kargoapi.Freight
	for _, req := range requested {
		// Get Freight directly from the Warehouse if allowed.
		if req.Sources.Direct && req.Origin.Kind == kargoapi.FreightOriginKindWarehouse {
			var directFreight kargoapi.FreightList
			if err := r.client.List(
				ctx,
				&directFreight,
				client.InNamespace(stage.Namespace),
				client.MatchingFieldsSelector{
					Selector: fields.OneTermEqualSelector(indexer.FreightByWarehouseField, req.Origin.Name),
				},
			); err != nil {
				return nil, fmt.Errorf(
					"error listing Freight from %q in namespace %q: %w",
					req.Origin.String(),
					stage.Namespace,
					err,
				)
			}

			for _, f := range directFreight.Items {
				// TODO(hidde): It would be better to use a fields.AndSelectors
				// above in combination with a fields.OneTermNotEqualSelector
				// to filter out Freight that has already been verified in this
				// Stage.
				//
				// However, the fake client does not support != field selectors,
				// and we would need a "real" Kubernetes API server to test it.
				// Until we (finally) make use of testenv, this will have to do.
				if _, ok := f.Status.VerifiedIn[stage.Name]; ok {
					continue
				}
				availableFreight = append(availableFreight, f)
			}
		}

		// Get Freight verified in upstream Stages.
		for _, upstream := range req.Sources.Stages {
			var verifiedFreight kargoapi.FreightList
			if err := r.client.List(
				ctx,
				&verifiedFreight,
				client.InNamespace(stage.Namespace),
				client.MatchingFieldsSelector{
					Selector: fields.OneTermEqualSelector(indexer.FreightByVerifiedStagesField, upstream),
				},
			); err != nil {
				return nil, fmt.Errorf(
					"error listing Freight from %q in namespace %q: %w",
					upstream,
					stage.Namespace,
					err,
				)
			}

			for _, f := range verifiedFreight.Items {
				// TODO(hidde): It would be better to use a fields.AndSelectors
				// above in combination with a fields.OneTermNotEqualSelector
				// to filter out Freight that has already been verified in this
				// Stage.
				//
				// However, the fake client does not support != field selectors,
				// and we would need a "real" Kubernetes API server to test it.
				// Until we (finally) make use of testenv, this will have to do.
				if _, ok := f.Status.VerifiedIn[stage.Name]; ok {
					continue
				}
				availableFreight = append(availableFreight, f)
			}
		}
	}

	// As the same Freight may be available due to multiple reasons (e.g. direct
	// from Warehouse and verified in upstream Stages), we need to deduplicate
	// the list.
	slices.SortFunc(availableFreight, func(lhs, rhs kargoapi.Freight) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})
	availableFreight = slices.CompactFunc(availableFreight, func(lhs, rhs kargoapi.Freight) bool {
		return lhs.Name == rhs.Name
	})

	return availableFreight, nil
}

// verifyFreight marks the given Freight as verified in the given Stage. It
// records an event for each Freight that is successfully verified.
func (r *ControlFlowStageReconciler) verifyFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
	freight []kargoapi.Freight,
	startTime, finishTime time.Time,
) error {
	logger := logging.LoggerFromContext(ctx)

	var failures int
	for _, f := range freight {
		// Skip Freight that has already been verified in this Stage.
		if _, ok := f.Status.VerifiedIn[stage.Name]; ok {
			continue
		}

		// Verify the Freight.
		newStatus := f.Status.DeepCopy()
		if newStatus.VerifiedIn == nil {
			newStatus.VerifiedIn = map[string]kargoapi.VerifiedStage{}
		}
		newStatus.VerifiedIn[stage.Name] = kargoapi.VerifiedStage{}
		if err := kubeclient.PatchStatus(ctx, r.client, &f, func(status *kargoapi.FreightStatus) {
			*status = *newStatus
		}); err != nil {
			if client.IgnoreNotFound(err) != nil {
				logger.Error(
					err,
					"failed to mark Freight as verified in Stage",
					"freight", f.Name,
				)
				failures++
			}
			continue
		}

		// Record an event for the verification.
		r.eventRecorder.AnnotatedEventf(
			stage,
			map[string]string{
				kargoapi.AnnotationKeyEventActor:                  kargoapi.FormatEventControllerActor(r.cfg.Name()),
				kargoapi.AnnotationKeyEventProject:                stage.Namespace,
				kargoapi.AnnotationKeyEventStageName:              stage.Name,
				kargoapi.AnnotationKeyEventFreightAlias:           f.Alias,
				kargoapi.AnnotationKeyEventFreightName:            f.Name,
				kargoapi.AnnotationKeyEventFreightCreateTime:      f.CreationTimestamp.Format(time.RFC3339),
				kargoapi.AnnotationKeyEventVerificationStartTime:  startTime.Format(time.RFC3339),
				kargoapi.AnnotationKeyEventVerificationFinishTime: finishTime.Format(time.RFC3339),
			},
			corev1.EventTypeNormal,
			kargoapi.EventReasonFreightVerificationSucceeded,
			"Freight verification succeeded",
		)
	}

	if failures > 0 {
		// Return an error if any of the verifications failed.
		// This will cause the Stage to be requeued.
		return fmt.Errorf("failed to verify %d Freight", failures)
	}
	return nil
}

// handleDelete handles the deletion of the given control flow Stage. It clears
// the verification status of all Freight that have been verified in the Stage,
// the approval status of all Freight that have been approved for the Stage, and
// deletes all AnalysisRuns that are associated with the Stage.
//
// It returns an error aggregate of all errors that occurred during the deletion
// process.
func (r *ControlFlowStageReconciler) handleDelete(ctx context.Context, stage *kargoapi.Stage) error {
	// If the Stage does not have the finalizer, there is nothing to do.
	if !controllerutil.ContainsFinalizer(stage, kargoapi.FinalizerName) {
		return nil
	}

	// Clear the verification and approval status of all Freight that have been
	// verified or approved for the Stage, and delete all AnalysisRuns.
	toClear := []func(context.Context, *kargoapi.Stage) error{
		r.clearVerifications,
		r.clearApprovals,
		r.clearAnalysisRuns,
	}
	var errs []error
	for _, c := range toClear {
		if err := c(ctx, stage); err != nil {
			errs = append(errs, err)
		}
	}
	if err := kerrors.Flatten(kerrors.NewAggregate(errs)); err != nil {
		// We ran into an error, return it before proceeding with removing the
		// finalizer.
		return fmt.Errorf("error handling deletion of Stage: %w", err)
	}

	// Remove the finalizer from the Stage.
	if err := kargoapi.RemoveFinalizer(ctx, r.client, stage); err != nil {
		return fmt.Errorf("error removing finalizer from Stage: %w", err)
	}

	return nil
}

// clearVerifications clears the verification status of all Freight that have
// been verified in the given Stage. It removes the Stage from the VerifiedIn
// map of each Freight.
func (r *ControlFlowStageReconciler) clearVerifications(ctx context.Context, stage *kargoapi.Stage) error {
	verified := kargoapi.FreightList{}
	if err := r.client.List(
		ctx,
		&verified,
		client.InNamespace(stage.Namespace),
		client.MatchingFieldsSelector{
			Selector: fields.OneTermEqualSelector(
				indexer.FreightByVerifiedStagesField,
				stage.Name,
			),
		},
	); err != nil {
		return fmt.Errorf(
			"error listing Freight verified in Stage %q in namespace %q: %w",
			stage.Name,
			stage.Namespace,
			err,
		)
	}

	var errs []error
	for _, f := range verified.Items {
		newStatus := *f.Status.DeepCopy()
		if newStatus.VerifiedIn == nil {
			continue
		}
		delete(newStatus.VerifiedIn, stage.Name)

		if err := kubeclient.PatchStatus(ctx, r.client, &f, func(status *kargoapi.FreightStatus) {
			*status = newStatus
		}); client.IgnoreNotFound(err) != nil {
			errs = append(errs, fmt.Errorf(
				"error clearing verification status of Freight %q in namespace %q: %w",
				f.Name, f.Namespace, err,
			))
		}
	}
	return kerrors.NewAggregate(errs)
}

// clearApprovals clears the approval status of all Freight that have been
// approved for the given Stage. It removes the Stage from the ApprovedFor map
// of each Freight.
func (r *ControlFlowStageReconciler) clearApprovals(ctx context.Context, stage *kargoapi.Stage) error {
	approved := kargoapi.FreightList{}
	if err := r.client.List(
		ctx,
		&approved,
		client.InNamespace(stage.Namespace),
		client.MatchingFieldsSelector{
			Selector: fields.OneTermEqualSelector(
				indexer.FreightApprovedForStagesField,
				stage.Name,
			),
		},
	); err != nil {
		return fmt.Errorf("error listing Freight approved for Stage %q in namespace %q: %w",
			stage.Name,
			stage.Namespace,
			err,
		)
	}

	var errs []error
	for _, f := range approved.Items {
		newStatus := *f.Status.DeepCopy()
		if newStatus.ApprovedFor == nil {
			continue
		}
		delete(newStatus.ApprovedFor, stage.Name)

		if err := kubeclient.PatchStatus(ctx, r.client, &f, func(status *kargoapi.FreightStatus) {
			*status = newStatus
		}); client.IgnoreNotFound(err) != nil {
			errs = append(errs, fmt.Errorf(
				"error clearing approval status of Freight %q in namespace %q: %w",
				f.Name, f.Namespace, err,
			))
		}
	}
	return kerrors.NewAggregate(errs)
}

// clearAnalysisRuns clears all AnalysisRuns that are associated with the given
// Stage. This is only done if the Rollouts integration is enabled.
func (r *ControlFlowStageReconciler) clearAnalysisRuns(ctx context.Context, stage *kargoapi.Stage) error {
	if !r.cfg.RolloutsIntegrationEnabled {
		return nil
	}

	if err := r.client.DeleteAllOf(
		ctx,
		&rollouts.AnalysisRun{},
		client.InNamespace(stage.Namespace),
		client.MatchingLabels(map[string]string{
			kargoapi.StageLabelKey: stage.Name,
		}),
	); err != nil {
		return fmt.Errorf("error deleting AnalysisRuns for Stage %q in namespace %q: %w",
			stage.Name,
			stage.Namespace,
			err,
		)
	}
	return nil
}
