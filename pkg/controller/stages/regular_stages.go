package stages

import (
	"context"
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/controller"
	argocdapi "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/metrics"
	kargoEvent "github.com/akuity/kargo/pkg/event"
	k8sevent "github.com/akuity/kargo/pkg/event/kubernetes"
	"github.com/akuity/kargo/pkg/health"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/kargo"
	"github.com/akuity/kargo/pkg/kubeclient"
	"github.com/akuity/kargo/pkg/kubernetes"
	libEvent "github.com/akuity/kargo/pkg/kubernetes/event"
	"github.com/akuity/kargo/pkg/logging"
	intpredicate "github.com/akuity/kargo/pkg/predicate"
)

// ReconcilerConfig represents configuration for the stage reconciler.
type ReconcilerConfig struct {
	IsDefaultController                bool   `envconfig:"IS_DEFAULT_CONTROLLER"`
	ShardName                          string `envconfig:"SHARD_NAME"`
	RolloutsIntegrationEnabled         bool   `envconfig:"ROLLOUTS_INTEGRATION_ENABLED"`
	RolloutsControllerInstanceID       string `envconfig:"ROLLOUTS_CONTROLLER_INSTANCE_ID"`
	MaxConcurrentControlFlowReconciles int    `envconfig:"MAX_CONCURRENT_CONTROL_FLOW_RECONCILES" default:"4"`
	MaxConcurrentReconciles            int    `envconfig:"MAX_CONCURRENT_STAGE_RECONCILES" default:"4"`
}

// Name returns the name of the Stage controller.
func (c ReconcilerConfig) Name() string {
	const name = "stage-controller"
	if c.ShardName != "" {
		return name + "-" + c.ShardName
	}
	return name
}

// ReconcilerConfigFromEnv returns a new ReconcilerConfig populated from the
// environment variables.
func ReconcilerConfigFromEnv() ReconcilerConfig {
	cfg := ReconcilerConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

type RegularStageReconciler struct {
	cfg            ReconcilerConfig
	client         client.Client
	eventSender    kargoEvent.Sender
	healthChecker  health.AggregatingChecker
	shardPredicate controller.ResponsibleFor[kargoapi.Stage]

	backoffCfg wait.Backoff
}

// NewRegularStageReconciler creates a new Stages reconciler.
func NewRegularStageReconciler(
	cfg ReconcilerConfig,
	healthChecker health.AggregatingChecker,
) *RegularStageReconciler {
	return &RegularStageReconciler{
		cfg:           cfg,
		healthChecker: healthChecker,
		shardPredicate: controller.ResponsibleFor[kargoapi.Stage]{
			IsDefaultController: cfg.IsDefaultController,
			ShardName:           cfg.ShardName,
		},
		backoffCfg: wait.Backoff{
			Duration: 1 * time.Second,
			Factor:   2,
			Steps:    10,
			Cap:      2 * time.Minute,
			Jitter:   0.1,
		},
	}
}

// SetupWithManager sets up the Stage reconciler with the given controller
// manager. It registers the reconciler with the manager and sets up watches
// on the required objects.
func (r *RegularStageReconciler) SetupWithManager(
	ctx context.Context,
	kargoMgr, argocdMgr ctrl.Manager,
	sharedIndexer client.FieldIndexer,
) error {
	// Configure client and event recorder using manager.
	r.client = kargoMgr.GetClient()
	r.eventSender = k8sevent.NewEventSender(
		libEvent.NewRecorder(ctx, kargoMgr.GetScheme(), kargoMgr.GetClient(), r.cfg.Name()),
	)

	// This index is used to find all Promotions that are associated with a
	// specific Stage.
	if err := sharedIndexer.IndexField(
		ctx,
		&kargoapi.Promotion{},
		indexer.PromotionsByStageField,
		indexer.PromotionsByStage,
	); err != nil {
		return fmt.Errorf("error setting up index for Promotions by Stage: %w", err)
	}

	// This index is used to determine if a Promotion already exists for a
	// Stage and Freight combination.
	if err := sharedIndexer.IndexField(
		ctx,
		&kargoapi.Promotion{},
		indexer.PromotionsByStageAndFreightField,
		indexer.PromotionsByStageAndFreight,
	); err != nil {
		return fmt.Errorf("error setting up index for Promotions by Stage and Freight: %w", err)
	}

	// This index is used to find Promotions that are non-terminal.
	if err := sharedIndexer.IndexField(
		ctx,
		&kargoapi.Promotion{},
		indexer.PromotionsByTerminalField,
		indexer.PromotionsByTerminal,
	); err != nil {
		return fmt.Errorf(
			"error setting up index for Promotions by terminal phase: %w", err,
		)
	}

	// This index is used to determine if a PromotionRequest already exists for a
	// Stage and Freight combination.
	if err := sharedIndexer.IndexField(
		ctx,
		&kargoapi.PromotionRequest{},
		indexer.PromotionRequestsByStageAndFreightField,
		indexer.PromotionRequestsByStageAndFreight,
	); err != nil {
		return fmt.Errorf(
			"error setting up index for PromotionRequests by Stage and Freight: %w", err,
		)
	}

	// This index is used to find all PromotionRequests that promote Freight on
	// behalf of a specific Stage.
	if err := sharedIndexer.IndexField(
		ctx,
		&kargoapi.PromotionRequest{},
		indexer.PromotionRequestsByStageField,
		indexer.PromotionRequestsByStage,
	); err != nil {
		return fmt.Errorf(
			"error setting up index for PromotionRequests by Stage: %w", err,
		)
	}

	// This index is used to find Freight that are directly available from a
	// Warehouse and can be automatically promoted to a Stage.
	if err := sharedIndexer.IndexField(
		ctx,
		&kargoapi.Freight{},
		indexer.FreightByWarehouseField,
		indexer.FreightByWarehouse,
	); err != nil {
		return fmt.Errorf("error setting up index for Freight by Warehouse: %w", err)
	}

	// This index is used to find all Freight that have been verified in upstream
	// Stages and can be automatically promoted to a Stage.
	if err := sharedIndexer.IndexField(
		ctx,
		&kargoapi.Freight{},
		indexer.FreightByVerifiedStagesField,
		indexer.FreightByVerifiedStages,
	); err != nil {
		return fmt.Errorf("error setting up index for Freight by Stages in which it has been verified: %w", err)
	}

	// This index is used to find all Freight that have been explicitly approved
	// for a Stage and can be automatically promoted to that Stage.
	if err := sharedIndexer.IndexField(
		ctx,
		&kargoapi.Freight{},
		indexer.FreightApprovedForStagesField,
		indexer.FreightApprovedForStages,
	); err != nil {
		return fmt.Errorf("index Freight by Stages for which it has been approved: %w", err)
	}

	metrics.RegisterStageMetrics(r.client)

	// Build the controller with the reconciler.
	c, err := ctrl.NewControllerManagedBy(kargoMgr).
		For(&kargoapi.Stage{}).
		WithEventFilter(controller.ResponsibleFor[client.Object]{
			IsDefaultController: r.cfg.IsDefaultController,
			ShardName:           r.cfg.ShardName,
		}).
		WithEventFilter(intpredicate.IgnoreDelete[client.Object]{}).
		WithEventFilter(
			predicate.And(
				IsControlFlowStage(false),
				predicate.Or(
					predicate.GenerationChangedPredicate{},
					kargo.RefreshRequested{},
					kargo.ReverifyRequested{},
					kargo.VerificationAbortRequested{},
				),
			),
		).
		WithOptions(controller.CommonOptions(r.cfg.MaxConcurrentReconciles)).
		Build(r)
	if err != nil {
		return fmt.Errorf("error building Stage reconciler: %w", err)
	}

	// Configure the watches.
	// Changes to these objects that match the constraints from the predicates
	// will enqueue a reconciliation for the related Stage(s).
	logger := logging.LoggerFromContext(ctx)

	// Watch for Promotions for which the phase changed and enqueue the related
	// Stage for reconciliation.
	if err = c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Promotion{},
			handler.TypedEnqueueRequestForOwner[*kargoapi.Promotion](
				kargoMgr.GetScheme(),
				kargoMgr.GetRESTMapper(),
				&kargoapi.Stage{},
				handler.OnlyControllerOwner(),
			),
			kargo.NewPromoPhaseChangedPredicate(logger),
		),
	); err != nil {
		return fmt.Errorf("unable to watch Promotions: %w", err)
	}

	// Watch for PromotionRequests for which the phase changed and enqueue the
	// related Stage for reconciliation.
	if err = c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.PromotionRequest{},
			handler.TypedEnqueueRequestForOwner[*kargoapi.PromotionRequest](
				kargoMgr.GetScheme(),
				kargoMgr.GetRESTMapper(),
				&kargoapi.Stage{},
				handler.OnlyControllerOwner(),
			),
			kargo.NewPromotionRequestPhaseChangedPredicate(logger),
		),
	); err != nil {
		return fmt.Errorf("unable to watch PromotionRequests: %w", err)
	}

	// Watch for Freight that have been newly promoted to a Stage or newly marked
	// as verified in a Stage and enqueue downstream Stages for reconciliation.
	if err = c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Freight{},
			&downstreamStageEnqueuer[*kargoapi.Freight]{
				kargoClient: kargoMgr.GetClient(),
			},
		),
	); err != nil {
		return fmt.Errorf("unable to watch Freight from upstream Stages: %w", err)
	}

	// Watch for Freight that has been approved for a Stage and enqueue the Stage
	// for reconciliation.
	if err = c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Freight{},
			&stageEnqueuerForApprovedFreight[*kargoapi.Freight]{
				kargoClient: kargoMgr.GetClient(),
			},
		),
	); err != nil {
		return fmt.Errorf("unable to watch approved Freight: %w", err)
	}

	// Watch for newly produced Freight from a Warehouse and enqueue the related
	// Stages for reconciliation.
	if err = c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Freight{},
			&warehouseStageEnqueuer[*kargoapi.Freight]{
				kargoClient: kargoMgr.GetClient(),
			},
		),
	); err != nil {
		return fmt.Errorf("unable to watch Freight produced by Warehouse: %w", err)
	}

	// If we have an ArgoCD manager, then we should watch for changes to
	// ArgCD Applications and enqueue the related Stages for reconciliation.
	if argocdMgr != nil {
		if err = c.Watch(
			source.Kind(
				argocdMgr.GetCache(),
				&argocdapi.Application{},
				&stageEnqueuerForArgoCDChanges[*argocdapi.Application]{
					kargoClient: kargoMgr.GetClient(),
				},
			),
		); err != nil {
			return fmt.Errorf("unable to watch Applications: %w", err)
		}
	}

	// If the Argo Rollouts integration is enabled, then we should watch for
	// changes to AnalysisRuns and enqueue the related Stages for reconciliation.
	if r.cfg.RolloutsIntegrationEnabled {
		if err = sharedIndexer.IndexField(
			ctx,
			&kargoapi.Stage{},
			indexer.StagesByAnalysisRunField,
			indexer.StagesByAnalysisRun(r.cfg.ShardName, r.cfg.IsDefaultController),
		); err != nil {
			return fmt.Errorf("error setting up index for Stages by AnalysisRun: %w", err)
		}

		if err = c.Watch(
			source.Kind(
				kargoMgr.GetCache(),
				&rolloutsapi.AnalysisRun{},
				&stageEnqueuerForAnalysisRuns[*rolloutsapi.AnalysisRun]{
					kargoClient: kargoMgr.GetClient(),
				},
			),
		); err != nil {
			return fmt.Errorf("unable to watch AnalysisRuns: %w", err)
		}
	}

	logging.LoggerFromContext(ctx).Info(
		"Initialized regular Stage reconciler",
		"maxConcurrentReconciles", r.cfg.MaxConcurrentControlFlowReconciles,
	)

	return nil
}

func (r *RegularStageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"namespace", req.Namespace,
		"stage", req.Name,
		"controlFlow", false,
	)
	ctx = logging.ContextWithLogger(ctx, logger)

	// Find the Stage.
	stage := &kargoapi.Stage{}
	if err := r.client.Get(ctx, req.NamespacedName, stage); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Safety check: do not reconcile Stages that are control flow Stages.
	if stage.IsControlFlow() {
		return ctrl.Result{}, nil
	}

	if !r.shardPredicate.IsResponsible(stage) {
		logger.Debug("ignoring Stage because it is not assigned to this shard")
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

	// Reconcile the Stage.
	logger.Debug("reconciling Stage")
	newStatus, needsRequeue, reconcileErr := r.reconcile(ctx, stage, time.Now())
	logger.Debug("done reconciling Stage")

	// Record the current refresh token as having been handled.
	if token, ok := api.RefreshAnnotationValue(stage.GetAnnotations()); ok {
		newStatus.LastHandledRefresh = token
	}

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

	// Return the reconcile error if it exists.
	if reconcileErr != nil {
		return ctrl.Result{}, reconcileErr
	}
	// Immediate requeue if needed.
	if needsRequeue {
		return ctrl.Result{RequeueAfter: 100 * time.Millisecond}, nil
	}
	// Otherwise, requeue after a delay.
	// TODO: Make the requeue delay configurable.
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *RegularStageReconciler) reconcile(
	ctx context.Context,
	stage *kargoapi.Stage,
	startTime time.Time,
) (kargoapi.StageStatus, bool, error) {
	logger := logging.LoggerFromContext(ctx)

	// working is the Stage the sub-reconcilers operate on. Its status is
	// unconditionally brought up to date after each sub-reconciler, whether or
	// not persisting that status succeeded, so no sub-reconciler ever computes
	// from a stale view of this pass.
	//
	// stage itself is only ever touched by PatchStatus, which refreshes it from
	// the server's response on success and leaves it alone on failure. It
	// therefore always reflects persisted state. Because it is also the base
	// PatchStatus diffs against, a failed patch loses nothing: the next patch
	// (including the final one in Reconcile) diffs against true server state and
	// re-carries any unpersisted changes.
	working := stage.DeepCopy()

	newStatus := *working.Status.DeepCopy()

	// Mark the Stage as reconciling.
	conditions.Set(&newStatus, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReconciling,
		Status:             metav1.ConditionTrue,
		Reason:             "Reconciling",
		ObservedGeneration: working.Generation,
	})

	// Determine once whether auto-promotion is enabled for the Stage, so every
	// sub-reconciler that depends on it sees a single, consistent answer for this
	// reconciliation. Reading it more than once risks different answers at
	// different points as the underlying ProjectConfig changes.
	autoPromotionEnabled, err := api.IsAutoPromotionEnabled(
		ctx,
		r.client,
		stage.ObjectMeta,
	)
	if err != nil {
		return newStatus, false, fmt.Errorf(
			"error checking auto-promotion policy for Stage %q: %w", stage.Name, err,
		)
	}

	var requestRequeue bool
	subReconcilers := []struct {
		name      string
		reconcile func() (kargoapi.StageStatus, error)
	}{
		{
			name: "syncing Promotions",
			reconcile: func() (kargoapi.StageStatus, error) {
				status, hasNonTerminalPromotions, err := r.syncPromotions(
					ctx,
					working,
					autoPromotionEnabled,
				)
				if err != nil {
					err = fmt.Errorf("failed to sync Promotions: %w", err)
				}
				// If we have no current Promotion and there are pending Promotions,
				// then we should request an immediate requeue to ensure that we
				// process the next Promotion as soon as possible.
				if getCurrentPromoObjectFromStatus(stage, status) == nil && hasNonTerminalPromotions {
					requestRequeue = true
				}
				return status, err
			},
		},
		{
			name: "syncing Freight",
			reconcile: func() (kargoapi.StageStatus, error) {
				if err := r.syncFreight(ctx, working); err != nil {
					return working.Status, fmt.Errorf("failed to sync Freight: %w", err)
				}
				return working.Status, nil
			},
		},
		{
			name: "assessing health",
			reconcile: func() (kargoapi.StageStatus, error) {
				status := r.assessHealth(ctx, working)
				// Unknown health is a normal, expected transient state (e.g. waiting
				// for Argo CD to reconcile after an operation). The Application watcher
				// will re-enqueue the Stage when the relevant condition changes, so
				// there is no longer a need to return an error here just for the sake
				// of triggering progressive backoff, as we did once upon a time.
				return status, nil
			},
		},
		{
			name: "verifying Stage Freight",
			reconcile: func() (kargoapi.StageStatus, error) {
				status, err := r.verifyStageFreight(ctx, working, startTime, time.Now)
				if err != nil {
					err = fmt.Errorf("failed to verify Stage Freight: %w", err)
				}
				// If we have a non-terminal verification for the current Freight,
				// then we should rely on the watcher to requeue the Stage when the
				// verification completes.
				curFreightCol := status.FreightHistory.Current()
				if curFreightCol != nil && curFreightCol.HasNonTerminalVerification() {
					requestRequeue = false
				}
				return status, err
			},
		},
		{
			name: "verifying Freight for Stage",
			reconcile: func() (kargoapi.StageStatus, error) {
				status, err := r.markFreightVerifiedForStage(ctx, working)
				if err != nil {
					err = fmt.Errorf("failed to verify Freight for Stage: %w", err)
				}
				return status, err
			},
		},
		{
			// This step must run immediately before "auto-promoting Freight":
			// it computes the effective hold state from the live Promotions at
			// the moment of the auto-promotion decision, which that next step
			// reads. Inserting a step between the two would let the effective
			// holds go stale relative to the decision.
			name: "computing effective auto-promotion holds",
			reconcile: func() (kargoapi.StageStatus, error) {
				status := *working.Status.DeepCopy()
				holds, err := r.computeEffectiveAutoPromotionHolds(
					ctx,
					working,
					autoPromotionEnabled,
				)
				if err != nil {
					return status, fmt.Errorf(
						"failed to compute effective auto-promotion holds: %w", err,
					)
				}
				status.EffectiveAutoPromotionHolds = holds
				return status, nil
			},
		},
		{
			name: "auto-promoting Freight",
			reconcile: func() (kargoapi.StageStatus, error) {
				status, err := r.autoPromoteFreight(ctx, working, autoPromotionEnabled)
				if err != nil {
					err = fmt.Errorf("failed to auto-promote Freight: %w", err)
				}
				return status, err
			},
		},
	}
	for _, subR := range subReconcilers {
		logger.Debug(subR.name)

		// Reconcile the Stage with the sub-reconciler.
		var err error
		newStatus, err = subR.reconcile()

		// Summarize the conditions after each sub-reconciler to ensure that
		// we have a consistent view of the Stage status.
		summarizeConditions(working, &newStatus, err)

		// If an error occurred during the sub-reconciler, then we should
		// return the error which will cause the Stage to be requeued.
		if err != nil {
			return newStatus, false, err
		}

		// Patch the status of the Stage after each sub-reconciler to show progress.
		// Failure is non-fatal: working carries this pass's status forward
		// regardless, and the next patch attempt's diff against stage (which still
		// reflects persisted state) will include these changes.
		if err = kubeclient.PatchStatus(ctx, r.client, stage, func(status *kargoapi.StageStatus) {
			*status = newStatus
		}); err != nil {
			logger.Error(err, fmt.Sprintf("failed to update Stage status after %s", subR.name))
		}
		working.Status = newStatus
	}

	// If an immediate requeue was not requested, then we can delete the
	// Reconciling condition as we have finished reconciling the Stage
	// and did not encounter any errors.
	if !requestRequeue {
		conditions.Delete(&newStatus, kargoapi.ConditionTypeReconciling)
	}

	return newStatus, requestRequeue, nil
}

// syncFreight ensures that all Freight statuses accurately reflect whether they
// are currently in use by the Stage.
func (r *RegularStageReconciler) syncFreight(ctx context.Context, stage *kargoapi.Stage) error {
	// Get the Stage's current FreightCollection.
	curFreight := stage.Status.FreightHistory.Current()
	// Find all Freight that think they're currently in use by this Stage.
	var freight []kargoapi.Freight
	freight, err := api.ListFreightByCurrentStage(ctx, r.client, stage)
	if err != nil {
		return err
	}
	// Step through all the Freight that think they're currently used by this
	// Stage and, if they're not, patch their status to accurately reflect that.
	for _, f := range freight {
		if !curFreight.Includes(f.Name) {
			newStatus := f.Status.DeepCopy()
			newStatus.RemoveCurrentStage(stage.Name)
			if err := kubeclient.PatchStatus(ctx, r.client, &f, func(status *kargoapi.FreightStatus) {
				*status = *newStatus
			}); err != nil {
				return fmt.Errorf(
					"error patching status of Freight %q in namespace %q: %w",
					f.Name, f.Namespace, err,
				)
			}
		}
	}
	// Iterate over every piece of Freight that the Stage is actually using to
	// make sure that their status accurately reflects that.
	//
	// This is the timestamp we'll use to track when the Freight came into use
	// by the Stage. There are edge cases where this won't be perfectly accurate,
	// but it's close enough, especially given that we use it for calculating
	// "soak time," which requires a certain MINIMUM amount of time to pass. Any
	// inaccuracy in the timestamp only means the Freight has actually "soaked"
	// LONGER than what we calculate.
	now := time.Now()
	for _, fr := range curFreight.References() {
		f, err := api.GetFreight(
			ctx,
			r.client,
			types.NamespacedName{
				Namespace: stage.Namespace,
				Name:      fr.Name,
			},
		)
		if err != nil {
			return fmt.Errorf(
				"error getting Freight %q in namespace %q: %w",
				fr.Name, stage.Namespace, err,
			)
		}
		if f == nil {
			// nolint:staticcheck
			return fmt.Errorf("Freight %q not found in namespace %q", fr.Name, stage.Namespace)
		}
		if !f.IsCurrentlyIn(stage.Name) {
			newStatus := f.Status.DeepCopy()
			newStatus.AddCurrentStage(stage.Name, now)
			if err = kubeclient.PatchStatus(ctx, r.client, f, func(status *kargoapi.FreightStatus) {
				*status = *newStatus
			}); err != nil {
				return fmt.Errorf(
					"error patching status of Freight %q in namespace %q: %w",
					f.Name, f.Namespace, err,
				)
			}
		}
	}
	return nil
}

// handleDelete handles the deletion of the given Stage. It clears the
// verification status of all Freight that have been verified in the Stage, the
// approval status of all Freight that have been approved for the Stage, and
// deletes all AnalysisRuns that are associated with the Stage.
//
// It returns an error aggregate of all errors that occurred during the deletion
// process.
func (r *RegularStageReconciler) handleDelete(ctx context.Context, stage *kargoapi.Stage) error {
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
	if err := api.RemoveFinalizer(ctx, r.client, stage); err != nil {
		return fmt.Errorf("error removing finalizer from Stage: %w", err)
	}

	return nil
}

// clearVerifications clears the verification status of all Freight that have
// been verified in the given Stage. It removes the Stage from the VerifiedIn
// map of each Freight.
func (r *RegularStageReconciler) clearVerifications(ctx context.Context, stage *kargoapi.Stage) error {
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
func (r *RegularStageReconciler) clearApprovals(ctx context.Context, stage *kargoapi.Stage) error {
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
func (r *RegularStageReconciler) clearAnalysisRuns(ctx context.Context, stage *kargoapi.Stage) error {
	if !r.cfg.RolloutsIntegrationEnabled {
		return nil
	}

	if err := r.client.DeleteAllOf(
		ctx,
		&rolloutsapi.AnalysisRun{},
		client.InNamespace(stage.Namespace),
		client.MatchingLabels(map[string]string{
			kargoapi.LabelKeyStage: kubernetes.ShortenLabelValue(stage.Name),
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

// summarizeConditions summarizes the conditions of the given Stage. It sets the
// Ready condition based on the Promoting, Healthy, and Verified conditions.
// If there is an error, the Ready condition is set to False until the error is
// resolved.
func summarizeConditions(stage *kargoapi.Stage, newStatus *kargoapi.StageStatus, err error) {
	// If there is an error, then we are not Ready until the error is resolved.
	if err != nil {
		conditions.Set(newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             "ReconcileError",
			Message:            err.Error(),
			ObservedGeneration: stage.Generation,
		})

		conditions.Set(newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeReconciling,
			Status:             metav1.ConditionTrue,
			Reason:             "RetryAfterError",
			ObservedGeneration: stage.Generation,
		})
		return
	}

	// Set the Freight summary.
	newStatus.FreightSummary = buildFreightSummary(len(stage.Spec.RequestedFreight), newStatus.FreightHistory.Current())

	// If we are currently Promoting, then we are not Ready.
	promoCond := conditions.Get(newStatus, kargoapi.ConditionTypePromoting)
	if promoCond != nil {
		conditions.Set(newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             promoCond.Reason,
			Message:            promoCond.Message,
			ObservedGeneration: stage.Generation,
		})
		return
	}

	// NOTE: this is the only place in the code where we get LastPromotion from newStatus and not original stage
	lastPromo := getLastPromoObjectFromStatus(stage, *newStatus)
	// If we are not currently Promoting but the last promotion failed,
	// then we are not Ready.
	if lastPromo != nil && lastPromo.GetPhase().IsTerminal() && !lastPromo.GetPhase().IsSucceeded() {
		conditions.Set(newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             fmt.Sprintf("LastPromotion%s", lastPromo.GetPhase().String()),
			Message:            lastPromo.GetMessage(),
			ObservedGeneration: stage.Generation,
		})
		return
	}

	// If we are not Healthy, then we are not Ready.
	healthCond := conditions.Get(newStatus, kargoapi.ConditionTypeHealthy)
	if healthCond == nil || healthCond.Status != metav1.ConditionTrue {
		readyCond := &metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             "Unhealthy",
			Message:            "Stage is not healthy",
			ObservedGeneration: stage.Generation,
		}
		if healthCond != nil {
			readyCond.Reason = healthCond.Reason
			readyCond.Message = healthCond.Message
		}
		conditions.Set(newStatus, readyCond)
		return
	}

	// If we are not verified, then we are not Ready.
	verificationCond := conditions.Get(newStatus, kargoapi.ConditionTypeVerified)
	if verificationCond == nil || verificationCond.Status != metav1.ConditionTrue {
		readyCond := &metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             "PendingVerification",
			Message:            "Stage is not verified",
			ObservedGeneration: stage.Generation,
		}
		if verificationCond != nil {
			readyCond.Reason = verificationCond.Reason
			readyCond.Message = verificationCond.Message
		}
		conditions.Set(newStatus, readyCond)
		return
	}

	// At this point, we can propagate the Ready condition from the Verified
	// condition.
	conditions.Set(newStatus, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		Reason:             verificationCond.Reason,
		Message:            verificationCond.Message,
		ObservedGeneration: stage.Generation,
	})
	conditions.Delete(newStatus, kargoapi.ConditionTypeReconciling)

	// If we are Ready, then we can also mark the current generation as
	// observed.
	newStatus.ObservedGeneration = stage.Generation
}

func buildFreightSummary(requested int, current *kargoapi.FreightCollection) string {
	if current == nil {
		return fmt.Sprintf("0/%d Fulfilled", requested)
	}
	if requested == 1 && len(current.Freight) == 1 {
		for _, f := range current.Freight {
			return f.Name
		}
	}
	return fmt.Sprintf("%d/%d Fulfilled", len(current.Freight), requested)
}
