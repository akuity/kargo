package stages

import (
	"context"
	"fmt"
	"slices"
	"strings"
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
				if status.CurrentPromotion == nil && hasNonTerminalPromotions {
					requestRequeue = true
				}
				return status, err
			},
		},
		{
			name: "syncing PromotionRequests",
			reconcile: func() (kargoapi.StageStatus, error) {
				status, err := r.syncPromotionRequests(ctx, working)
				if err != nil {
					err = fmt.Errorf("failed to sync PromotionRequests: %w", err)
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

// withoutTargetPromotions removes Promotions to one of the Stage's Targets (the
// children of a PromotionRequest) from a list of the Stage's Promotions, and
// returns what remains. Children take no part in the Stage's own promotion
// flow: their admission is decided by the Stage's current PromotionRequest --
// see api.StageAwaitsPromotion -- and their outcomes are the business of
// whatever governs the Target, so the Stage records nothing about them.
// Every place the Stage reconciler lists its own Promotions filters through
// this, so that the invariant holds structurally rather than by accident of
// which annotations or code paths children happen to reach.
//
// spec.target is a sound discriminator because admission enforces it: the
// Promotion webhook rejects a Promotion that names a Target but is not owned
// by a PromotionRequest.
func withoutTargetPromotions(promos []kargoapi.Promotion) []kargoapi.Promotion {
	return slices.DeleteFunc(promos, func(promo kargoapi.Promotion) bool {
		return promo.Spec.Target != ""
	})
}

func (r *RegularStageReconciler) getPromotions(
	ctx context.Context,
	stage kargoapi.Stage,
) ([]kargoapi.Promotion, error) {
	// List all Promotions for the Stage.
	promotions := &kargoapi.PromotionList{}
	if err := r.client.List(
		ctx,
		promotions,
		client.InNamespace(stage.Namespace),
		client.MatchingFieldsSelector{
			Selector: fields.OneTermEqualSelector(indexer.PromotionsByStageField, stage.Name),
		},
	); err != nil {
		return nil, fmt.Errorf(
			"failed to list Promotions for Stage %q in namespace %q: %w",
			stage.Name, stage.Namespace, err,
		)
	}
	return promotions.Items, nil
}

// syncPromotionRequests records in the Stage's status which PromotionRequest is
// currently fanning Freight out to the Stage's Targets, and which was the last
// to reach a terminal phase.
//
// The current reference is the mutex that serializes rounds of fan-out: the
// Promotion reconciler runs a Promotion to one of the Stage's Targets only
// while the PromotionRequest that owns it is the Stage's current one -- see
// api.StageAwaitsPromotion. All of one request's children (at most one per
// Target) are admitted at once, so Promotions to distinct Targets run in
// parallel, while the children of a queued request wait for the current
// round to end. The last reference remains a mirror, kept so that a reader
// of the Stage can see how the previous round ended without listing
// PromotionRequests.
//
// A Stage can have more than one PromotionRequest in flight, exactly as it can
// have more than one Promotion in flight: auto-promotion creates a request only
// when none exists in any phase, but the promote endpoints create one per call,
// so consecutive promotions queue up. Which one the Stage records as current is
// therefore decided by the same ordering syncPromotions applies to Promotions.
//
// The references mirror the requests that exist, not the Stage's spec: a
// request in flight for a Stage whose selectors currently govern no Targets
// -- or one left behind by a Stage that no longer governs any -- is recorded
// all the same. Only for a Stage with no PromotionRequests at all is this a
// no-op beyond clearing a stale current reference.
func (r *RegularStageReconciler) syncPromotionRequests(
	ctx context.Context,
	stage *kargoapi.Stage,
) (kargoapi.StageStatus, error) {
	newStatus := *stage.Status.DeepCopy()

	promotionRequests := &kargoapi.PromotionRequestList{}
	if err := r.client.List(
		ctx,
		promotionRequests,
		client.InNamespace(stage.Namespace),
		client.MatchingFieldsSelector{
			Selector: fields.OneTermEqualSelector(
				indexer.PromotionRequestsByStageField,
				stage.Name,
			),
		},
	); err != nil {
		return newStatus, fmt.Errorf(
			"failed to list PromotionRequests for Stage %q in namespace %q: %w",
			stage.Name, stage.Namespace, err,
		)
	}

	// If there are no PromotionRequests, the Stage is fanning nothing out. Clear
	// any current reference it was left with.
	if len(promotionRequests.Items) == 0 {
		newStatus.CurrentPromotionRequest = nil
		return newStatus, nil
	}

	// Sort the PromotionRequests exactly as syncPromotions sorts a Stage's
	// Promotions -- Running first, then non-terminal by ULID ascending, then
	// terminal by ULID descending -- so that the request a Stage records as
	// current is chosen the same way its current Promotion is.
	slices.SortFunc(
		promotionRequests.Items,
		api.ComparePromotionRequestByPhaseAndCreationTime,
	)

	// The PromotionRequest with the highest priority is the one the Stage is
	// promoting through, unless it has finished -- in which case the Stage is
	// promoting through none, and a finished request must not be left looking
	// like an active one.
	newStatus.CurrentPromotionRequest = nil
	if highestPrioRequest := &promotionRequests.Items[0]; !highestPrioRequest.Status.Phase.IsTerminal() {
		newStatus.CurrentPromotionRequest = newPromotionRequestReference(highestPrioRequest)
	}

	// Gather the terminal PromotionRequests newer than the one already recorded
	// as last. A request is recorded only when it is newer: a Stage's account of
	// how its last round of fan-out ended should outlive the request that
	// produced it, so garbage collection of the newest request must not let an
	// older one take its place.
	//
	// Every such request is gathered, not just the newest. More than one round
	// can end between two reconciles -- a queued request failing while the round
	// ahead of it succeeds, or several rounds ending while the controller was
	// down -- and each succeeded round's Freight belongs in the Stage's history.
	// Recording only the newest would drop the others for good, since the gate
	// only moves forward by name.
	//
	// NB: As in syncPromotions, this makes use of the fact that PromotionRequest
	// names are generated with an embedded ULID, so among one Stage's requests
	// lex order over names is creation order.
	var newRequests []*kargoapi.PromotionRequest
	for i := range promotionRequests.Items {
		promotionRequest := &promotionRequests.Items[i]
		if !promotionRequest.Status.Phase.IsTerminal() {
			continue
		}
		if last := newStatus.LastPromotionRequest; last != nil &&
			strings.Compare(promotionRequest.Name, last.Name) <= 0 {
			// Terminal PromotionRequests sort newest-first, so nothing after this
			// one is newer than the last recorded either.
			break
		}
		newRequests = append(newRequests, promotionRequest)
	}

	// Replay them oldest-first, exactly as syncPromotions replays Promotions, so
	// that the last reference lands on the newest and Freight history is
	// recorded in the order the rounds ended. Each record builds on the status
	// the one before it produced, which is what lets a multi-origin Stage's
	// collection carry one round's Freight into the next.
	slices.SortFunc(newRequests, func(a, b *kargoapi.PromotionRequest) int {
		return strings.Compare(a.Name, b.Name)
	})
	//
	// A request is recorded before it becomes the last: the status is persisted
	// even when this returns an error, and a request that had already moved the
	// gate forward when fetching its Freight failed would never be considered
	// again, leaving its Freight out of the history for good. Recording first
	// leaves the gate on the previous request, so the next reconcile retries.
	for _, promotionRequest := range newRequests {
		if err := r.recordSucceededPromotionRequest(
			ctx,
			stage,
			&newStatus,
			promotionRequest,
		); err != nil {
			return newStatus, err
		}
		newStatus.LastPromotionRequest = newPromotionRequestReference(promotionRequest)
	}

	return newStatus, nil
}

// recordSucceededPromotionRequest records the Freight a succeeded
// PromotionRequest promoted as the Stage's current Freight, exactly as
// syncPromotions records a succeeded Promotion's. A Stage that promotes through
// PromotionRequests has no other writer of its freight history: its Promotions
// are children of a request and take no part in its own flow.
//
// Only a request that succeeded -- every Target's child Promotion succeeded --
// is recorded; a partial round leaves the Stage's account of what it is running
// unchanged, as a failed Promotion does. The caller invokes this once per
// request, at the moment the request is first recorded as the Stage's last,
// which is what keeps a request from being recorded again on every reconcile:
// freight history is prepend-only, and recording also resets health and
// verification.
//
// The collection is built here, at the moment of recording, from the Freight
// the request names and whatever the Stage is running now. Building it any
// earlier -- when the request is created, say -- would snapshot the Stage's
// other origins before a queued-ahead request had finished changing them, and
// recording that snapshot later would quietly roll those origins back. This is
// the same guarantee a Promotion gets from having its collection built only
// once it is admitted to run.
func (r *RegularStageReconciler) recordSucceededPromotionRequest(
	ctx context.Context,
	stage *kargoapi.Stage,
	newStatus *kargoapi.StageStatus,
	promotionRequest *kargoapi.PromotionRequest,
) error {
	if promotionRequest.Status.Phase != kargoapi.PromotionRequestPhaseSucceeded {
		return nil
	}
	logger := logging.LoggerFromContext(ctx).WithValues(
		"promotionRequest", promotionRequest.Name,
		"freight", promotionRequest.Spec.Freight,
	)

	freight, err := api.GetFreight(ctx, r.client, types.NamespacedName{
		Namespace: stage.Namespace,
		Name:      promotionRequest.Spec.Freight,
	})
	if err != nil {
		return fmt.Errorf(
			"error getting Freight %q promoted by PromotionRequest %q: %w",
			promotionRequest.Spec.Freight, promotionRequest.Name, err,
		)
	}
	if freight == nil {
		// Gone before the Stage could record it. Nothing is invented; the
		// Stage's account of what it is running is simply left as it was.
		logger.Debug("Freight promoted by succeeded PromotionRequest no longer exists: not recording it")
		return nil
	}

	// Inherit from the status being built, not the one the Stage was read with:
	// they hold the same history, but the former is what the Stage is about to
	// declare it is running.
	working := *stage
	working.Status = *newStatus
	newStatus.FreightHistory.Record(
		api.NewFreightCollectionForStage(&working, kargoapi.FreightReference{
			Name:      freight.Name,
			Commits:   freight.Commits,
			Images:    freight.Images,
			Charts:    freight.Charts,
			Artifacts: freight.Artifacts,
			Origin:    freight.Origin,
		}),
	)

	// The Stage is running new Freight: what was known about the health and
	// verification of the old is no longer relevant.
	newStatus.Health = nil
	conditions.Set(newStatus, &metav1.Condition{
		Type:               kargoapi.ConditionTypeHealthy,
		Status:             metav1.ConditionUnknown,
		Reason:             "WaitingForHealthCheck",
		Message:            "Waiting for health check to be performed after successful promotion",
		ObservedGeneration: stage.Generation,
	})
	conditions.Set(newStatus, &metav1.Condition{
		Type:               kargoapi.ConditionTypeVerified,
		Status:             metav1.ConditionUnknown,
		Reason:             "WaitingForVerification",
		Message:            "Waiting for verification to be performed after successful promotion",
		ObservedGeneration: stage.Generation,
	})
	return nil
}

// newPromotionRequestReference builds the reference a Stage records for one of
// its PromotionRequests. The reference names the PromotionRequest's Freight
// rather than describing it; a reader that needs the Freight's contents can
// look them up from the Freight itself.
func newPromotionRequestReference(
	promotionRequest *kargoapi.PromotionRequest,
) *kargoapi.PromotionRequestReference {
	return &kargoapi.PromotionRequestReference{
		Name:       promotionRequest.Name,
		Phase:      promotionRequest.Status.Phase,
		FinishedAt: promotionRequest.Status.FinishedAt,
		Freight: &kargoapi.PromotionRequestFreightReference{
			Name: promotionRequest.Spec.Freight,
		},
	}
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

	// If we are not currently Promoting but the last promotion failed,
	// then we are not Ready.
	if lastPromo := newStatus.LastPromotion; lastPromo != nil && lastPromo.Status != nil &&
		lastPromo.Status.Phase.IsTerminal() && lastPromo.Status.Phase != kargoapi.PromotionPhaseSucceeded {
		conditions.Set(newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             fmt.Sprintf("LastPromotion%s", string(lastPromo.Status.Phase)),
			Message:            lastPromo.Status.Message,
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
