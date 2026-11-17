package stages

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/client/apiserver"
	"github.com/akuity/kargo/pkg/client/apiserver/freightclient"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/controller"
	"github.com/akuity/kargo/pkg/controller/stages/verification"
	"github.com/akuity/kargo/pkg/event"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/kargo"
	"github.com/akuity/kargo/pkg/kubeclient"
	"github.com/akuity/kargo/pkg/logging"
	intpredicate "github.com/akuity/kargo/pkg/predicate"
	"github.com/akuity/kargo/pkg/telemetry"
)

type ControlFlowStageReconciler struct {
	cfg            ReconcilerConfig
	client         client.Client
	apiClient      *apiserver.Client
	eventSender    event.Sender
	shardPredicate controller.ResponsibleFor[kargoapi.Stage]

	// freightClient reads and writes Freight status through the control plane's
	// API server rather than the Kubernetes client. It is a field so tests can
	// substitute a fake; the constructor wires it to the API server client.
	freightClient freightclient.FreightClient
}

// NewControlFlowStageReconciler returns a new control flow Stage reconciler.
// After creating the reconciler, call SetupWithManager to register it with a
// controller manager.
func NewControlFlowStageReconciler(
	cfg ReconcilerConfig,
	apiClient *apiserver.Client,
) *ControlFlowStageReconciler {
	r := &ControlFlowStageReconciler{
		cfg:       cfg,
		apiClient: apiClient,
		shardPredicate: controller.ResponsibleFor[kargoapi.Stage]{
			IsDefaultController: cfg.IsDefaultController,
			ShardName:           cfg.ShardName,
		},
	}
	if apiClient != nil {
		r.freightClient = apiClient
	}
	return r
}

// SetupWithManager sets up the control flow Stage reconciler with the given
// controller manager. It registers the reconciler with the manager and sets up
// watches on the required objects.
func (r *ControlFlowStageReconciler) SetupWithManager(
	ctx context.Context,
	mgr ctrl.Manager,
	sharedIndexer client.FieldIndexer,
	eventSender event.Sender,
	natsConn *nats.Conn,
) error {
	// The reconciler reads and writes Freight status through the API server
	// rather than Kubernetes, so an API server client is required.
	if r.apiClient == nil {
		return errors.New(
			"the control flow Stage reconciler requires an API server client; " +
				"set API_SERVER_ADDRESS",
		)
	}

	// Configure client using manager.
	r.client = mgr.GetClient()
	r.eventSender = eventSender

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
		WithEventFilter(controller.ResponsibleFor[client.Object]{
			IsDefaultController: r.cfg.IsDefaultController,
			ShardName:           r.cfg.ShardName,
		}).
		WithEventFilter(intpredicate.IgnoreDelete[client.Object]{}).
		WithEventFilter(
			predicate.And(
				IsControlFlowStage(true),
				predicate.Or(
					predicate.GenerationChangedPredicate{},
					kargo.RefreshRequested{},
				),
			),
		).
		WithOptions(controller.CommonOptions(r.cfg.MaxConcurrentControlFlowReconciles)).
		Build(r)
	if err != nil {
		return fmt.Errorf("error building control flow Stage reconciler: %w", err)
	}

	// Configure the watches.
	// Changes to these objects that match the constraints from the predicates
	// will enqueue a reconciliation for the related Stage(s).
	//
	// The Freight signals this reconciler reacts to are delivered as Kargo
	// CloudEvents over NATS and consumed via source.Channel watches, mirroring the
	// regular Stage reconciler, rather than through Kubernetes Freight watches.

	// Watch for newly produced Freight from a Warehouse (FreightCreated events)
	// and enqueue the subscribed control flow Stages for reconciliation.
	freightCreatedCh, err := subscribeFreightCreated(ctx, natsConn)
	if err != nil {
		return fmt.Errorf("unable to subscribe to FreightCreated events: %w", err)
	}
	if err = c.Watch(
		source.Channel(
			freightCreatedCh,
			NewFreightCreatedStageEnqueuer(mgr.GetClient(), true),
		),
	); err != nil {
		return fmt.Errorf("unable to watch Freight produced by Warehouse: %w", err)
	}

	// Watch for Freight newly verified in an upstream Stage
	// (FreightVerifiedInStage events) and enqueue downstream control flow Stages
	// for reconciliation.
	verifiedCh, err := subscribeFreightVerifiedInStage(ctx, natsConn)
	if err != nil {
		return fmt.Errorf("unable to subscribe to FreightVerifiedInStage events: %w", err)
	}
	if err = c.Watch(
		source.Channel(
			verifiedCh,
			NewFreightVerifiedStageEnqueuer(mgr.GetClient(), true),
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
func (r *ControlFlowStageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, err error) {
	ctx, span := startReconcileSpan(ctx, req)
	defer func() {
		telemetry.RecordError(span, err)
		span.End()
	}()

	logger := logging.LoggerFromContext(ctx).WithValues(
		"namespace", req.Namespace,
		"stage", req.Name,
		"controlFlow", true,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	// Find the Stage.
	stage := &kargoapi.Stage{}
	if err := r.client.Get(ctx, req.NamespacedName, stage); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !r.shardPredicate.IsResponsible(stage) {
		logger.Debug("ignoring Control Flow Stage because it is not assigned to this shard")
		return ctrl.Result{}, nil
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
	if ok, err := api.EnsureFinalizer(ctx, r.client, stage); ok || err != nil {
		return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, err
	}

	// Remove any stale annotations from the Stage which are not relevant to
	// a control flow Stage.
	if stage.GetAnnotations()[kargoapi.AnnotationKeyArgoCDContext] != "" {
		if err := api.AnnotateStageWithArgoCDContext(
			ctx,
			r.client,
			nil,
			client.ObjectKeyFromObject(stage),
		); err != nil {
			logger.Error(err, "failed to remove Argo CD context annotation from Stage")
		}
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
	freight, err := r.listAvailableFreight(ctx, stage)
	if err != nil {
		conditions.Set(
			&newStatus,
			&metav1.Condition{
				Type:    kargoapi.ConditionTypeReconciling,
				Status:  metav1.ConditionTrue,
				Reason:  "RetryAfterFreightRetrievalFailed",
				Message: err.Error(),
			},
			&metav1.Condition{
				Type:    kargoapi.ConditionTypeReady,
				Status:  metav1.ConditionFalse,
				Reason:  "FreightRetrievalFailed",
				Message: err.Error(),
			},
		)
		return newStatus, err
	}

	err = UnmarkUnusedFreight(ctx, r.freightClient, stage, func(fr kargoapi.Freight) bool {
		for _, available := range freight {
			if available.Name == fr.Name {
				return false
			}
		}
		return true
	})
	if err != nil {
		conditions.Set(
			&newStatus,
			&metav1.Condition{
				Type:    kargoapi.ConditionTypeReconciling,
				Status:  metav1.ConditionTrue,
				Reason:  "RetryAfterUnmarkingFreightFailed",
				Message: err.Error(),
			},
			&metav1.Condition{
				Type:    kargoapi.ConditionTypeReady,
				Status:  metav1.ConditionFalse,
				Reason:  "UnmarkingFreightFailed",
				Message: err.Error(),
			},
		)
		return newStatus, err
	}

	// If there is new Freight to verify, do so.
	if len(freight) > 0 {
		newlyVerified, err := r.markFreightVerifiedForStage(ctx, stage, freight, startTime, time.Now())
		if newlyVerified > 0 {
			logger.Debug("verified Freight", "count", newlyVerified)
		}
		if err != nil {
			conditions.Set(
				&newStatus,
				&metav1.Condition{
					Type:    kargoapi.ConditionTypeReconciling,
					Status:  metav1.ConditionTrue,
					Reason:  "RetryAfterVerificationFailed",
					Message: err.Error(),
				},
				&metav1.Condition{
					Type:    kargoapi.ConditionTypeReady,
					Status:  metav1.ConditionFalse,
					Reason:  "FreightVerificationFailed",
					Message: err.Error(),
				},
			)
			return newStatus, err
		}
	}

	// Mark the Stage as Ready and remove any reconciling condition.
	conditions.Set(&newStatus, &metav1.Condition{
		Type:   kargoapi.ConditionTypeReady,
		Status: metav1.ConditionTrue,
		Reason: kargoapi.ConditionTypeReady,
	})
	conditions.Delete(&newStatus, kargoapi.ConditionTypeReconciling)

	return newStatus, nil
}

// listAvailableFreight lists the Freight available to the Stage, reading through
// the API server rather than the Kubernetes client.
func (r *ControlFlowStageReconciler) listAvailableFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
) ([]kargoapi.Freight, error) {
	return api.ListFreightAvailableToStageViaClient(ctx, r.freightClient, stage)
}

// initializeStatus initializes the status of the given Stage with the values
// that are common to all control flow Stages. It resets the status to a clean
// state, recording the current refresh token as having been handled.
func (r *ControlFlowStageReconciler) initializeStatus(stage *kargoapi.Stage) kargoapi.StageStatus {
	newStatus := stage.Status.DeepCopy()

	// Update the status with the new observed generation and phase.
	if stage.Generation > stage.Status.ObservedGeneration {
		newStatus.ObservedGeneration = stage.Generation
	}

	// Record the current refresh token as having been handled.
	if token, ok := api.RefreshAnnotationValue(stage.GetAnnotations()); ok {
		newStatus.LastHandledRefresh = token
	}

	// Only keep the conditions that are relevant to this Stage type.
	var condCopy []metav1.Condition
	for _, c := range []string{kargoapi.ConditionTypeReady, kargoapi.ConditionTypeReconciling} {
		if cond := conditions.Get(newStatus, c); cond != nil {
			condCopy = append(condCopy, *cond)
		}
	}
	newStatus.Conditions = condCopy

	// Clear all the fields that are not relevant to this Stage type.
	newStatus.FreightHistory = nil
	newStatus.Health = nil
	newStatus.CurrentPromotion = nil
	newStatus.LastPromotion = nil
	newStatus.FreightSummary = "N/A"
	newStatus.AutoPromotionEnabled = false
	// Holds left behind by a regular Stage converted to control flow would
	// otherwise be orphaned: only the regular Stage reconciler transitions or
	// removes them.
	newStatus.AutoPromotionHolds = nil

	return *newStatus
}

// markFreightVerifiedForStage marks the given Freight as verified in the given
// Stage, unless it already has been. It records an event for each Freight newly
// marked as verified and returns the total number of Freight that were
// marked as verified.
func (r *ControlFlowStageReconciler) markFreightVerifiedForStage(
	ctx context.Context,
	stage *kargoapi.Stage,
	freight []kargoapi.Freight,
	startTime, finishTime time.Time,
) (int, error) {
	logger := logging.LoggerFromContext(ctx)

	var newlyVerified int
	var failures int
	for _, f := range freight {
		// Skip Freight that has already been verified in this Stage.
		//
		// TODO(hidde + krancour): It would be better to filter out Freight that has
		// already been verified in this Stage at retrieval time, but the fake
		// client does not support != field selectors, so we would need a "real"
		// Kubernetes API server to test it. Until we (finally) make use of testenv,
		// this will have to do.
		if f.IsVerifiedIn(stage.Name) {
			continue
		}

		// Mark the Freight as verified in the Stage through the API server rather
		// than writing Freight status directly. verified=true records it in both
		// currentlyIn and verifiedIn and makes the API server emit the
		// corresponding events.
		if err := r.freightClient.MarkFreightInStage(
			ctx, stage.Namespace, f.Name, stage.Name, true, finishTime,
		); err != nil {
			// Freight deleted out from under us is not a failure; it simply no
			// longer needs verifying.
			if apiserver.IsNotFound(err) {
				continue
			}
			logger.Error(
				err,
				"failed to mark Freight as verified in Stage",
				"freight", f.Name,
			)
			failures++
			continue
		}

		newlyVerified++

		evt, err := event.NewFreightVerification(
			api.FormatEventControllerActor(r.cfg.Name()),
			stage.Name,
			&f,
			&kargoapi.VerificationInfo{
				Phase:      kargoapi.VerificationPhaseSuccessful,
				StartTime:  &metav1.Time{Time: startTime},
				FinishTime: &metav1.Time{Time: finishTime},
			},
			nil,
		)
		if err == nil {
			err = r.eventSender.Send(ctx, evt)
		}
		if err != nil {
			logger.Error(
				err,
				"failed to send Freight verification succeeded event",
			)
		}
	}

	if failures > 0 {
		// Return an error if any of the verifications failed.
		// This will cause the Stage to be requeued.
		return newlyVerified, fmt.Errorf("failed to verify %d Freight", failures)
	}
	return newlyVerified, nil
}

// handleDelete handles the deletion of the given control flow Stage. It clears
// the verification status of all Freight that have been verified in the Stage,
// the approval status of all Freight that have been approved for the Stage, and
// deletes all AnalysisRuns that are associated with the Stage.
//
// It returns an error aggregate of all errors that occurred during the deletion
// process.
func (r *ControlFlowStageReconciler) handleDelete(ctx context.Context, stage *kargoapi.Stage) error {
	return HandleDelete(ctx, r.newVerifier(), r.client, r.freightClient, stage)
}

// Instatinate verifier similar to regular stages in case stage transitions from regular to control flow.
func (r *ControlFlowStageReconciler) newVerifier() verification.Verifier {
	analysisRunners := map[kargoapi.AnalysisRunKind]verification.AnalysisRunner{
		kargoapi.AnalysisRunKindRun: verification.NewAnalysisRunnerRollouts(
			r.cfg.RolloutsControllerInstanceID,
			wait.Backoff{},
			r.client,
			nil,
			r.cfg.RolloutsIntegrationEnabled,
		),
	}
	ver := verification.NewVerifier(
		r.cfg.Name(),
		r.client,
		r.freightClient,
		r.eventSender,
		analysisRunners,
		nil,
	)
	return ver
}
