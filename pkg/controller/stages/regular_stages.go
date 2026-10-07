package stages

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	"github.com/akuity/kargo/pkg/controller/stages/verification"
	"github.com/akuity/kargo/pkg/credentials"
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
	"github.com/akuity/kargo/pkg/telemetry"
)

// ReconcilerConfig represents configuration for the stage reconciler.
type ReconcilerConfig struct {
	IsDefaultController                bool   `envconfig:"IS_DEFAULT_CONTROLLER"`
	ShardName                          string `envconfig:"SHARD_NAME"`
	RolloutsIntegrationEnabled         bool   `envconfig:"ROLLOUTS_INTEGRATION_ENABLED"`
	RolloutsControllerInstanceID       string `envconfig:"ROLLOUTS_CONTROLLER_INSTANCE_ID"`
	MaxConcurrentControlFlowReconciles int    `envconfig:"MAX_CONCURRENT_CONTROL_FLOW_RECONCILES" default:"4"`
	MaxConcurrentReconciles            int    `envconfig:"MAX_CONCURRENT_STAGE_RECONCILES" default:"4"`
	MaxConcurrentFleetReconciles       int    `envconfig:"MAX_CONCURRENT_FLEET_STAGE_RECONCILES" default:"4"`
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
	credentialsDB  credentials.Database
	eventSender    kargoEvent.Sender
	healthChecker  health.AggregatingChecker
	shardPredicate controller.ResponsibleFor[kargoapi.Stage]

	backoffCfg wait.Backoff
}

// NewRegularStageReconciler creates a new Stages reconciler.
func NewRegularStageReconciler(
	cfg ReconcilerConfig,
	credentialsDB credentials.Database,
	healthChecker health.AggregatingChecker,
) *RegularStageReconciler {
	return &RegularStageReconciler{
		cfg:           cfg,
		credentialsDB: credentialsDB,
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
				IsTargetAwareStage(false),
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
		"maxConcurrentReconciles", r.cfg.MaxConcurrentReconciles,
	)

	return nil
}

// tracer is the instrumentation scope under which this package's spans are
// recorded.
var tracer = otel.Tracer("github.com/akuity/kargo/pkg/controller/stages")

func (r *RegularStageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, span := tracer.Start(
		ctx,
		"Reconcile Stage",
		trace.WithAttributes(
			telemetry.ProjectKey.String(req.Namespace),
			telemetry.StageKey.String(req.Name),
		),
	)
	defer span.End()

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

	// Safety check: do not reconcile fleet Stages.
	if stage.IsTargetAware() {
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
				status, hasPendingPromotions, err := r.syncPromotions(
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
				if status.CurrentPromotion == nil && hasPendingPromotions {
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
		if summarizeConditions(stage, &newStatus, err) {
			// If we are Ready, then we can also mark the current generation as
			// observed.
			newStatus.ObservedGeneration = stage.Generation
		}

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

// syncPromotions synchronizes the Promotions for a Stage. It determines the
// current state of the Stage based on the Promotions that are running or have
// completed.
func (r *RegularStageReconciler) syncPromotions(
	ctx context.Context,
	stage *kargoapi.Stage,
	autoPromotionEnabled bool,
) (kargoapi.StageStatus, bool, error) {
	logger := logging.LoggerFromContext(ctx)
	newStatus := *stage.Status.DeepCopy()

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
		err = fmt.Errorf(
			"failed to list Promotions for Stage %q in namespace %q: %w",
			stage.Name, stage.Namespace, err,
		)

		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypePromoting,
			Status:             metav1.ConditionUnknown,
			Reason:             "ListPromotionsFailed",
			Message:            err.Error(),
			ObservedGeneration: stage.Generation,
		})

		return newStatus, false, err
	}

	// Build a map of origin keys that are currently requested by this Stage,
	// used both to filter new holds and to evict stale ones.
	requestedOrigins := make(map[string]struct{}, len(stage.Spec.RequestedFreight))
	for _, req := range stage.Spec.RequestedFreight {
		requestedOrigins[req.Origin.String()] = struct{}{}
	}

	// While auto-promotion is disabled, holds are meaningless: the Stage's
	// current Freight is held in place by auto-promotion being disabled, not by
	// the holds. Clear them so that re-enabling auto-promotion is a uniform fresh
	// start, consistent with the fact that promoting non-candidate Freight while
	// auto-promotion is disabled never establishes a hold. (Hold establishment in
	// the replay below is gated on autoPromotionEnabled too.)
	if !autoPromotionEnabled {
		newStatus.AutoPromotionHolds = nil
	} else {
		// Drop holds for origins that are no longer requested. This runs before
		// the early-exit below so it takes effect even when there are no
		// Promotions.
		for key := range newStatus.AutoPromotionHolds {
			if _, ok := requestedOrigins[key]; !ok {
				delete(newStatus.AutoPromotionHolds, key)
			}
		}
		if len(newStatus.AutoPromotionHolds) == 0 {
			newStatus.AutoPromotionHolds = nil
		}
	}

	// If there are no Promotions, then we are not promoting any Freight.
	if len(promotions.Items) == 0 {
		logger.Debug("no Promotions found for Stage")

		// Ensure we delete any existing "current" Promotion related information
		// from the Stage status.
		conditions.Delete(&newStatus, kargoapi.ConditionTypePromoting)
		newStatus.CurrentPromotion = nil

		return newStatus, false, nil
	}

	// Sort the Promotions by phase and creation time to determine the current
	// state of the Stage.
	slices.SortFunc(promotions.Items, api.ComparePromotionByPhaseAndCreationTime)

	// The Promotion with the highest priority (i.e. a Running or Pending phase)
	// is the one that we will consider for the current state of the Stage.
	highestPrioPromo := promotions.Items[0]

	// The Promotion which is currently running on the Stage.
	currentPromo := stage.Status.CurrentPromotion

	// The last Promotion which ran on the Stage.
	lastPromo := stage.Status.LastPromotion

	// Track if there are any non-terminal promotions that need handling.
	// This is later used to determine if we should issue an immediate
	// requeue.
	var hasNonTerminalPromotions bool
	for _, promo := range promotions.Items {
		if !promo.Status.Phase.IsTerminal() {
			hasNonTerminalPromotions = true
			break
		}
	}

	// If the current Promotion is not the highest priority Promotion, or the
	// highest priority Promotion is in a terminal phase, then we must have
	// finished promoting.
	if currentPromo != nil && (currentPromo.Name != highestPrioPromo.Name || highestPrioPromo.Status.Phase.IsTerminal()) {
		// Update the conditions to reflect that we are no longer promoting.
		conditions.Delete(&newStatus, kargoapi.ConditionTypePromoting)
		newStatus.CurrentPromotion = nil

		// Gather terminal Promotions newer than the last processed one, sorted
		// oldest-to-newest so holds are applied in chronological order and
		// Freight history entries are appended oldest-first (GC removes oldest
		// first).
		var newPromos []*kargoapi.Promotion
		for i := range promotions.Items {
			promo := &promotions.Items[i]
			if lastPromo != nil {
				// We can break here since we know that all subsequent Promotions
				// will be older than the last Promotion we saw.
				// NB: This makes use of the fact that Promotion names are
				// generated, and contain a timestamp component which will ensure
				// that they can be sorted in a consistent order.
				if strings.Compare(promo.Name, lastPromo.Name) <= 0 {
					break
				}
			}
			if promo.Status.Phase.IsTerminal() {
				newPromos = append(newPromos, promo)
			}
		}
		slices.SortFunc(newPromos, func(a, b *kargoapi.Promotion) int {
			return strings.Compare(a.Name, b.Name)
		})

		// Replay new Promotions in chronological order to update hold state and
		// Stage status. Hold/release intent is applied in order so a later
		// release correctly supersedes an earlier hold and vice versa. Holds
		// persist in status even after their establishing Promotion is GC'd.
		for _, promo := range newPromos {
			// A Promotion's hold/resume intent is fixed at creation and is not
			// changed by an involuntary failure, so any terminal Promotion
			// applies its intent. The exception is an Aborted Promotion: the
			// user deliberately canceled it, withdrawing the intent along with
			// it. (newPromos contains only terminal Promotions.) Holds are only
			// maintained while auto-promotion is enabled; when disabled they are
			// cleared above and not re-established here.
			if autoPromotionEnabled && promo.Status.Phase != kargoapi.PromotionPhaseAborted {
				if originKey := promo.Annotations[kargoapi.AnnotationKeyAutoPromotionHold]; originKey != "" {
					if _, requested := requestedOrigins[originKey]; requested {
						if origin, err := kargoapi.ParseFreightOrigin(originKey); err == nil {
							if newStatus.AutoPromotionHolds == nil {
								newStatus.AutoPromotionHolds = make(map[string]kargoapi.AutoPromotionHold)
							}
							newStatus.AutoPromotionHolds[originKey] = newAutoPromotionHold(promo, origin)
						}
					}
				} else if originKey := promo.Annotations[kargoapi.AnnotationKeyAutoPromotionResume]; originKey != "" {
					delete(newStatus.AutoPromotionHolds, originKey)
				}
			}
			ref := kargoapi.PromotionReference{
				Name:       promo.Name,
				Status:     promo.Status.DeepCopy(),
				FinishedAt: promo.Status.FinishedAt,
			}
			if promo.Status.Freight != nil {
				ref.Freight = promo.Status.Freight.DeepCopy()
			}
			// A Promotion that was aborted before it ever reached Running never
			// had the chance to build a FreightCollection. Recording it as-is
			// would make the Stage forget Freight origins the previous
			// lastPromotion had already collected, permanently breaking any
			// subsequent Promotion's ability to inherit them.
			if promo.Status.StartedAt == nil && ref.Status.FreightCollection == nil &&
				newStatus.LastPromotion != nil && newStatus.LastPromotion.Status != nil {
				ref.Status.FreightCollection = newStatus.LastPromotion.Status.FreightCollection
			}
			newStatus.LastPromotion = &ref
			if promo.Status.Phase == kargoapi.PromotionPhaseSucceeded {
				// If the Promotion was successful, then we should add the Freight
				// to the history of successfully promoted Freight.
				newStatus.FreightHistory.Record(ref.Status.FreightCollection)

				// Erase any health checks that were performed for the previous
				// Freight, as they are no longer relevant.
				newStatus.Health = nil
				conditions.Set(&newStatus, &metav1.Condition{
					Type:               kargoapi.ConditionTypeHealthy,
					Status:             metav1.ConditionUnknown,
					Reason:             "WaitingForHealthCheck",
					Message:            "Waiting for health check to be performed after successful promotion",
					ObservedGeneration: stage.Generation,
				})

				// Set verified condition to unknown to indicate that the
				// new Freight needs to be verified.
				conditions.Set(&newStatus, &metav1.Condition{
					Type:               kargoapi.ConditionTypeVerified,
					Status:             metav1.ConditionUnknown,
					Reason:             "WaitingForVerification",
					Message:            "Waiting for verification to be performed after successful promotion",
					ObservedGeneration: stage.Generation,
				})

				// Annotate the Stage with the latest information related to
				// ArgoCD Applications. This is used to provide deep links to the
				// ArgoCD UI for the Stage in the Kargo UI.
				//
				// NB: If the Promotion did not involve any ArgoCD Applications,
				// then the annotation will be removed.
				if err := api.AnnotateStageWithArgoCDContext(
					ctx,
					r.client,
					promo,
					client.ObjectKeyFromObject(stage),
				); err != nil {
					// Let the error be logged, but do not return it as it is not
					// critical to the operation of the Stage.
					logger.Error(err, "failed to annotate Stage with ArgoCD context")
				}
			}
		}

		// Return at this point to allow the new Freight to be verified.
		return newStatus, hasNonTerminalPromotions, nil
	}

	// If the current Freight exists and has a non-terminal verification, wait
	// for it to complete regardless of health state to ensure we capture the
	// results.
	if curFreight := stage.Status.FreightHistory.Current(); curFreight != nil {
		if curFreight.HasNonTerminalVerification() {
			logger.Debug(
				"current Freight has a non-terminal verification: " +
					"wait for it to complete before allowing new promotions to start",
			)
			conditions.Delete(&newStatus, kargoapi.ConditionTypePromoting)
			return newStatus, hasNonTerminalPromotions, nil
		}

		// If we are in a healthy state, the current Freight needs to be verified
		// before we can allow the next Promotion to start. If we are unhealthy
		// or the verification failed, then we can allow the next Promotion to
		// start immediately as the expectation is that the Promotion can fix the
		// issue.
		if stage.Status.Health == nil || stage.Status.Health.Status == kargoapi.HealthStateHealthy {
			curVI := curFreight.VerificationHistory.Current()
			if curVI == nil || !curVI.Phase.IsTerminal() {
				logger.Debug("current Freight needs to be verified before allowing new promotions to start")
				conditions.Delete(&newStatus, kargoapi.ConditionTypePromoting)
				return newStatus, hasNonTerminalPromotions, nil
			}
		}
	}

	// If the highest priority Promotion is not in a terminal phase, then we
	// are promoting the Freight.
	if !highestPrioPromo.Status.Phase.IsTerminal() {
		conditions.Set(&newStatus, &metav1.Condition{
			Type:   kargoapi.ConditionTypePromoting,
			Status: metav1.ConditionTrue,
			Reason: "ActivePromotion",
			Message: fmt.Sprintf(
				"Promotion %q is currently %s",
				highestPrioPromo.Name, highestPrioPromo.Status.Phase,
			),
			ObservedGeneration: stage.Generation,
		})

		newStatus.CurrentPromotion = &kargoapi.PromotionReference{
			Name: highestPrioPromo.Name,
		}
		if freight := highestPrioPromo.Status.Freight; freight != nil {
			newStatus.CurrentPromotion.Freight = freight.DeepCopy()
		}
		return newStatus, hasNonTerminalPromotions, nil
	}

	// If the highest priority Promotion is in a terminal phase, then we are
	// not promoting.
	conditions.Delete(&newStatus, kargoapi.ConditionTypePromoting)
	return newStatus, hasNonTerminalPromotions, nil
}

// assessHealth assesses the health of a Stage based on the health checks from
// the last Promotion.
func (r *RegularStageReconciler) assessHealth(ctx context.Context, stage *kargoapi.Stage) kargoapi.StageStatus {
	logger := logging.LoggerFromContext(ctx)
	newStatus := *stage.Status.DeepCopy()

	if currentPromo := stage.Status.CurrentPromotion; currentPromo != nil {
		logger.Debug("Promotion is in progress: no health checks to perform")
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionUnknown,
			Reason:             "ActivePromotion",
			Message:            "Stage has a Promotion in progress",
			ObservedGeneration: stage.Generation,
		})
		newStatus.Health = &kargoapi.Health{
			Status: kargoapi.HealthStateUnknown,
			Issues: []string{
				"Cannot assess health because a Promotion is currently in progress",
			},
		}
		return newStatus
	}

	lastPromo := stage.Status.LastPromotion
	if lastPromo == nil {
		logger.Debug("Stage has no current Freight: no health checks to perform")
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionUnknown,
			Reason:             "NoFreight",
			Message:            "Stage has no current Freight",
			ObservedGeneration: stage.Generation,
		})
		newStatus.Health = nil
		return newStatus
	}

	// If the last Promotion did not succeed, then we cannot perform any health
	// checks because they are only available after a successful Promotion.
	// Since we lack enough information to determine health, we mark it as Unknown.
	//
	// TODO(hidde): Long term, this should probably be changed to allow to
	//  continue to run health checks from the last successful Promotion,
	//  even if the current Promotion did not succeed (e.g. because it was
	//  aborted).
	if lastPromo.Status.Phase != kargoapi.PromotionPhaseSucceeded {
		logger.Debug("Last promotion did not succeed: defaulting Stage health to Unknown")
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionUnknown,
			Reason:             fmt.Sprintf("LastPromotion%s", lastPromo.Status.Phase),
			Message:            "Cannot assess health because last Promotion did not succeed",
			ObservedGeneration: stage.Generation,
		})
		newStatus.Health = &kargoapi.Health{
			Status: kargoapi.HealthStateUnknown,
			Issues: []string{"Cannot assess health because last Promotion did not succeed"},
		}
		return newStatus
	}

	// Compose the health check criteria.
	healthChecks := lastPromo.Status.HealthChecks
	var criteria []health.Criteria
	for _, check := range healthChecks {
		criteria = append(criteria, health.Criteria{
			Kind:  check.Uses,
			Input: check.GetConfig(),
		})
	}

	// Run the hlth checks.
	hlth := r.healthChecker.Check(ctx, stage.Namespace, stage.Name, criteria)
	newStatus.Health = &hlth

	// Set the Healthy condition based on the health status.
	switch hlth.Status {
	case kargoapi.HealthStateHealthy:
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionTrue,
			Reason:             string(hlth.Status),
			Message:            fmt.Sprintf("Stage is healthy (performed %d health checks)", len(healthChecks)),
			ObservedGeneration: stage.Generation,
		})
	case kargoapi.HealthStateUnhealthy:
		conditions.Set(&newStatus, &metav1.Condition{
			Type:   kargoapi.ConditionTypeHealthy,
			Status: metav1.ConditionFalse,
			Reason: string(hlth.Status),
			Message: fmt.Sprintf(
				"Stage is unhealthy (%d issues in %d health checks)",
				len(hlth.Issues), len(healthChecks),
			),
			ObservedGeneration: stage.Generation,
		})
	default:
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionUnknown,
			Reason:             string(hlth.Status),
			ObservedGeneration: stage.Generation,
		})
	}

	return newStatus
}

// syncFreight ensures that all Freight statuses accurately reflect whether they
// are currently in use by the Stage.
func (r *RegularStageReconciler) syncFreight(ctx context.Context, stage *kargoapi.Stage) error {
	return syncFreight(ctx, r.client, stage)
}

func syncFreight(ctx context.Context, cl client.Client, stage *kargoapi.Stage) error {
	// Get the Stage's current FreightCollection.
	curFreight := stage.Status.FreightHistory.Current()
	// Find all Freight that think they're currently in use by this Stage.
	var freight []kargoapi.Freight
	freight, err := api.ListFreightByCurrentStage(ctx, cl, stage)
	if err != nil {
		return err
	}
	// Step through all the Freight that think they're currently used by this
	// Stage and, if they're not, patch their status to accurately reflect that.
	for _, f := range freight {
		if !curFreight.Includes(f.Name) {
			newStatus := f.Status.DeepCopy()
			newStatus.RemoveCurrentStage(stage.Name)
			if err := kubeclient.PatchStatus(ctx, cl, &f, func(status *kargoapi.FreightStatus) {
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
			cl,
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
			if err = kubeclient.PatchStatus(ctx, cl, f, func(status *kargoapi.FreightStatus) {
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

// verifyStageFreight verifies the current Freight of a Stage. If the Stage has
// no current Freight, or the Freight has already been verified, then no action
// is taken. If the Freight has not been verified yet, then a new verification
// is started.
//
// An annotation can be set on the Stage to request the verification to be
// aborted. This is useful if the verification is taking too long, or if the
// Freight is no longer needed.
//
// In addition, an annotation can be set on the Stage to request re-verification
// of the Freight. This can be useful to ensure that the current Freight is
// still in a good state.
//
// When the Stage is unhealthy, or a Promotion is currently running, then the
// verification is skipped.
func (r *RegularStageReconciler) verifyStageFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
	startTime time.Time,
	endTime func() time.Time,
) (newStatus kargoapi.StageStatus, err error) {
	analysisRunners := map[kargoapi.AnalysisRunGVK]verification.AnalysisRunner{
		kargoapi.AnalysisRunGVKRun: verification.NewAnalysisRunnerRollouts(
			r.cfg.RolloutsControllerInstanceID,
			r.backoffCfg,
			r.client,
			r.credentialsDB,
		),
	}
	ver := verification.NewVerifier(
		r.cfg.Name(),
		r.cfg.RolloutsIntegrationEnabled,
		r.client,
		r.eventSender,
		analysisRunners,
	)
	return ver.VerifyStageFreight(ctx, stage, startTime, endTime)
}

// markFreightVerifiedForStage marks the Freight that is associated with the
// Stage as verified. If the Freight has already been verified, then no action
// is taken.
func (r *RegularStageReconciler) markFreightVerifiedForStage(
	ctx context.Context,
	stage *kargoapi.Stage,
) (kargoapi.StageStatus, error) {
	return markFreightVerifiedForStage(ctx, r.client, stage)
}

func markFreightVerifiedForStage(
	ctx context.Context,
	cl client.Client,
	stage *kargoapi.Stage,
) (kargoapi.StageStatus, error) {
	logger := logging.LoggerFromContext(ctx)
	newStatus := *stage.Status.DeepCopy()

	// If the Stage is unhealthy, then we should not verify the Freight.
	if stage.Status.Health == nil || stage.Status.Health.Status != kargoapi.HealthStateHealthy {
		return newStatus, nil
	}

	// If there is no current Freight, or the Stage has not been verified yet
	// after the last Promotion, then we are not ready to verify the Freight.
	curFreight := stage.Status.FreightHistory.Current()
	if curFreight == nil ||
		len(curFreight.VerificationHistory) == 0 ||
		curFreight.HasNonTerminalVerification() ||
		curFreight.VerificationHistory.Current().Phase != kargoapi.VerificationPhaseSuccessful {
		return newStatus, nil
	}

	// At this point, all preconditions for verifying the Freight have been met,
	// and we can proceed with the verification.
	for _, ref := range curFreight.Freight {
		freight := &kargoapi.Freight{}
		if err := cl.Get(ctx, types.NamespacedName{
			Namespace: stage.Namespace,
			Name:      ref.Name,
		}, freight); err != nil {
			return newStatus, fmt.Errorf(
				"error getting Freight %q in namespace %q: %w",
				ref.Name, stage.Namespace, err,
			)
		}

		// If the Freight has already been verified, then there is no need to
		// verify it again.
		if freight.IsVerifiedIn(stage.Name) {
			logger.Debug("Freight has already been verified in Stage")
			continue
		}

		// Verify the Freight.
		if err := kubeclient.PatchStatus(ctx, cl, freight, func(status *kargoapi.FreightStatus) {
			if status.VerifiedIn == nil {
				status.VerifiedIn = make(map[string]kargoapi.VerifiedStage)
			}
			status.AddVerifiedStage(stage.Name, curFreight.VerificationHistory.Current().FinishTime.Time)
		}); err != nil {
			return newStatus, fmt.Errorf(
				"error marking Freight %q as verified in Stage: %w",
				freight.Name, err,
			)
		}
		logger.Debug("marked Freight as verified in Stage", "freight", freight.Name)
	}

	return newStatus, nil
}

// newAutoPromotionHold builds an AutoPromotionHold for origin from the
// hold-intent Promotion promo.
func newAutoPromotionHold(
	promo *kargoapi.Promotion,
	origin kargoapi.FreightOrigin,
) kargoapi.AutoPromotionHold {
	hold := kargoapi.AutoPromotionHold{
		FreightName:   promo.Spec.Freight,
		Origin:        origin,
		PromotionName: promo.Name,
	}
	if actor := promo.Annotations[kargoapi.AnnotationKeyCreateActor]; actor != "" {
		hold.Actor = actor
	}
	if !promo.CreationTimestamp.IsZero() {
		t := promo.CreationTimestamp
		hold.CreatedAt = &t
	}
	return hold
}

// computeEffectiveAutoPromotionHolds returns the set of auto-promotion holds in
// effect for the Stage right now. It starts from the durable holds in
// Status.AutoPromotionHolds -- preserving a hold whose establishing Promotion
// has been garbage-collected -- and overlays the newest non-aborted intent for
// each requested origin: a hold-intent Promotion holds the origin, a
// release-intent Promotion clears it, and the newest of the two wins. It
// returns an empty map while auto-promotion is disabled, when holds are
// meaningless. It does not mutate stage.
func (r *RegularStageReconciler) computeEffectiveAutoPromotionHolds(
	ctx context.Context,
	stage *kargoapi.Stage,
	autoPromotionEnabled bool,
) (map[string]kargoapi.AutoPromotionHold, error) {
	if !autoPromotionEnabled {
		return nil, nil
	}

	effective := make(map[string]kargoapi.AutoPromotionHold, len(stage.Status.AutoPromotionHolds))
	for key, hold := range stage.Status.AutoPromotionHolds {
		effective[key] = hold
	}

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

	lastPromo := stage.Status.LastPromotion
	for _, req := range stage.Spec.RequestedFreight {
		originKey := req.Origin.String()
		var newest *kargoapi.Promotion
		var newestIsHold bool
		for i := range promotions.Items {
			promo := &promotions.Items[i]
			if promo.Status.Phase == kargoapi.PromotionPhaseAborted {
				continue
			}
			// Only Promotions newer than the last one syncPromotions recorded can
			// change the durable state. Older ones are already reflected in it, and
			// the Promotion that superseded them may since have been deleted.
			if lastPromo != nil && strings.Compare(promo.Name, lastPromo.Name) <= 0 {
				continue
			}
			isHold := promo.Annotations[kargoapi.AnnotationKeyAutoPromotionHold] == originKey
			isRelease := promo.Annotations[kargoapi.AnnotationKeyAutoPromotionResume] == originKey
			if !isHold && !isRelease {
				continue
			}
			if newest == nil || strings.Compare(promo.Name, newest.Name) > 0 {
				newest = promo
				newestIsHold = isHold
			}
		}
		switch {
		case newest == nil:
			// No in-flight intent for this origin; leave the durable state as-is.
		case newestIsHold:
			effective[originKey] = newAutoPromotionHold(newest, req.Origin)
		default:
			delete(effective, originKey)
		}
	}
	return effective, nil
}

// autoPromoteFreight automatically promotes the candidate Freight for each
// requested origin, unless auto-promotion is disabled or the origin has an
// effective auto-promotion hold.
func (r *RegularStageReconciler) autoPromoteFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
	autoPromotionEnabled bool,
) (kargoapi.StageStatus, error) {
	logger := logging.LoggerFromContext(ctx)
	newStatus := *stage.Status.DeepCopy()
	newStatus.AutoPromotionEnabled = autoPromotionEnabled

	// If the Stage has no requested Freight, then there is nothing to promote.
	// NB: This should not happen in practice, as a Stage cannot exist without
	// requested Freight.
	if len(stage.Spec.RequestedFreight) == 0 {
		return newStatus, nil
	}

	logger.Debug("checked auto-promotion policy for Stage", "enabled", autoPromotionEnabled)
	if !autoPromotionEnabled {
		// Nothing to promote. The durable and effective hold maps are cleared by
		// syncPromotions and computeEffectiveAutoPromotionHolds respectively when
		// auto-promotion is disabled, not here.
		return newStatus, nil
	}

	availableFreight, err := api.ListFreightAvailableToStage(ctx, r.client, stage)
	if err != nil {
		return newStatus, fmt.Errorf(
			"error listing available Freight for Stage %q: %w",
			stage.Name, err,
		)
	}
	candidates := api.SelectAutoPromotionCandidates(ctx, stage, availableFreight)
	// If the Stage has no current Freight, any candidate is new to it.
	currentFreight := newStatus.FreightHistory.Current()

	// Check if there is any new Freight which can be auto-promoted.
	for _, req := range stage.Spec.RequestedFreight {
		origin := req.Origin.String()
		// Never create an auto-promotion for an origin with an effective hold.
		if _, held := stage.Status.EffectiveAutoPromotionHolds[origin]; held {
			logger.Debug("auto-promotion is blocked by an auto-promotion hold", "origin", origin)
			continue
		}

		candidate, exists := candidates[origin]
		if !exists {
			logger.Debug("no Freight from origin available for auto-promotion", "origin", origin)
			continue
		}

		freightLogger := logger.WithValues("origin", origin, "freight", candidate.Name)

		// Only proceed if the candidate Freight is not already current in the Stage.
		if freightCollectionHasFreight(currentFreight, origin, candidate.Name) {
			freightLogger.Debug("Stage already has candidate Freight for origin")
			continue
		}
		if stageAwaitingFreightForOrigin(stage, origin, candidate.Name) {
			freightLogger.Debug("Stage is already awaiting candidate Freight for origin")
			continue
		}

		// Do not create duplicate work: stand down while any Promotion for
		// this candidate is either still in flight or succeeded with an
		// outcome not yet recorded in Stage status.
		var unprocessedPromotionExists bool
		unprocessedPromotionExists, err = r.unprocessedPromotionExistsForStageFreight(
			ctx,
			stage,
			candidate.Name,
		)
		if err != nil {
			return newStatus, fmt.Errorf(
				"error listing existing Promotions for Freight %q in namespace "+
					"%q: %w",
				candidate.Name, stage.Namespace, err,
			)
		}
		if unprocessedPromotionExists {
			freightLogger.Debug("an unprocessed Promotion already exists for " +
				"Stage and Freight")
			continue
		}

		var newestPromotion *kargoapi.Promotion
		newestPromotion, err = r.newestTerminalPromotionForStageFreight(
			ctx,
			stage,
			candidate.Name,
		)
		if err != nil {
			return newStatus, fmt.Errorf(
				"error listing existing terminal Promotions for Freight %q in "+
					"namespace %q: %w",
				candidate.Name, stage.Namespace, err,
			)
		}
		if newestPromotion != nil &&
			newestPromotion.Status.Phase != kargoapi.PromotionPhaseSucceeded {
			freightLogger.Debug(
				"most recent terminal Promotion for Stage and Freight was not "+
					"successful; skipping auto-promotion to avoid an infinite loop",
				"lastPromotion", newestPromotion.Name,
				"lastPromotionPhase", newestPromotion.Status.Phase,
			)
			continue
		}

		// Auto-promote the candidate Freight and record an event. Create a minimal
		// Promotion. The defaulting webhook fills in the rest from the Stage's
		// PromotionTemplate.
		promotion := api.NewMinimalPromotion(stage, candidate.Name)
		if err = r.client.Create(ctx, promotion); err != nil {
			// An admission webhook may deny the create. Tolerate this as a
			// non-error: nothing is persisted, so this reconcile moves on, and a
			// later one re-derives and re-attempts the auto-promotion once the
			// denying policy no longer applies. Any other error is still fatal to
			// the reconcile.
			//
			// Deliberately not recorded as an event. A policy that holds keeps
			// denying, so this branch is taken on every reconcile for as long as
			// it does; an event per occurrence would report one unchanging
			// condition thousands of times and bury the Project's event feed. A
			// condition belongs in status, and Kargo Enterprise reports the one
			// it knows about in Stage.status.promotionSchedule.
			if apierrors.IsForbidden(err) {
				freightLogger.Debug(
					"auto-promotion was denied by an admission webhook",
					"error", err.Error(),
				)
				continue
			}
			return newStatus, fmt.Errorf(
				"error creating Promotion for Freight %q in namespace %q: %w",
				candidate.Name, stage.Namespace, err,
			)
		}
		evt := kargoEvent.NewPromotionCreated(
			fmt.Sprintf("Automatically promoted Freight from origin %q for Stage %q",
				origin,
				promotion.Spec.Stage),
			api.FormatEventControllerActor(r.cfg.Name()),
			promotion,
			&candidate,
		)
		if err := r.eventSender.Send(ctx, evt); err != nil {
			logger.Error(err, "failed to send promotion event")
		}
		logger.Debug(
			"created Promotion resource",
			"promotion", promotion.Name,
		)
	}

	return newStatus, nil
}

// stageAwaitingFreightForOrigin reports whether this reconcile pass has already
// observed a Promotion for the named Freight and origin. autoPromoteFreight uses
// it to avoid creating a duplicate Promotion before the status patch from
// syncPromotions reaches the API server.
func stageAwaitingFreightForOrigin(
	stage *kargoapi.Stage,
	origin string,
	name string,
) bool {
	if stage.Status.CurrentPromotion == nil ||
		stage.Status.CurrentPromotion.Freight == nil {
		return false
	}
	// Reconcile patches stage.Status back onto the in-memory Stage after each
	// sub-reconciler, so this sees Promotions observed earlier in this pass.
	return stage.Status.CurrentPromotion.Freight.Name == name &&
		stage.Status.CurrentPromotion.Freight.Origin.String() == origin
}

// unprocessedPromotionExistsForStageFreight reports whether a Promotion for
// this Stage and Freight exists whose outcome syncPromotions has not yet
// recorded: one that is still non-terminal, or one that SUCCEEDED after this
// reconciliation's view of the Stage was computed (i.e., is newer than
// status.lastPromotion). autoPromoteFreight uses it to avoid creating
// duplicate work for the same candidate. The second case matters because a
// fast Promotion can go from pending to succeeded in the interval between
// syncPromotions observing it and autoPromoteFreight acting; a
// non-terminal-only check misses it, and a succeeded Promotion for the
// candidate is deliberately not otherwise disqualifying. Once the next
// reconciliation records the success, the candidate-is-already-current check
// takes over. Promotions that reached any other terminal phase are handled
// by the newest-terminal-not-successful check regardless of whether they
// have been recorded, so they never block here.
func (r *RegularStageReconciler) unprocessedPromotionExistsForStageFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
	freightName string,
) (bool, error) {
	promotions := &kargoapi.PromotionList{}
	if err := r.client.List(
		ctx,
		promotions,
		client.InNamespace(stage.Namespace),
		client.MatchingFieldsSelector{
			Selector: fields.OneTermEqualSelector(
				indexer.PromotionsByStageAndFreightField,
				indexer.StageAndFreightKey(stage.Name, freightName),
			),
		},
	); err != nil {
		return false, err
	}

	lastPromo := stage.Status.LastPromotion
	for i := range promotions.Items {
		promo := &promotions.Items[i]
		if !promo.Status.Phase.IsTerminal() ||
			(promo.Status.Phase == kargoapi.PromotionPhaseSucceeded &&
				(lastPromo == nil || strings.Compare(promo.Name, lastPromo.Name) > 0)) {
			return true, nil
		}
	}
	return false, nil
}

// newestTerminalPromotionForStageFreight returns the newest completed Promotion
// for this Stage and Freight. autoPromoteFreight uses it to avoid retrying
// terminal failures in a loop.
func (r *RegularStageReconciler) newestTerminalPromotionForStageFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
	freightName string,
) (*kargoapi.Promotion, error) {
	promotions := &kargoapi.PromotionList{}
	if err := r.client.List(
		ctx,
		promotions,
		client.InNamespace(stage.Namespace),
		client.MatchingFieldsSelector{
			Selector: fields.AndSelectors(
				fields.OneTermEqualSelector(
					indexer.PromotionsByStageAndFreightField,
					indexer.StageAndFreightKey(stage.Name, freightName),
				),
				fields.OneTermEqualSelector(
					indexer.PromotionsByTerminalField,
					strconv.FormatBool(true),
				),
			),
		},
	); err != nil {
		return nil, err
	}

	if len(promotions.Items) == 0 {
		return nil, nil
	}
	slices.SortFunc(promotions.Items, func(lhs, rhs kargoapi.Promotion) int {
		if result := rhs.CreationTimestamp.Compare(lhs.CreationTimestamp.Time); result != 0 {
			return result
		}
		return strings.Compare(rhs.Name, lhs.Name)
	})
	return &promotions.Items[0], nil
}

// freightCollectionHasFreight checks a single origin in a FreightCollection.
func freightCollectionHasFreight(
	collection *kargoapi.FreightCollection,
	origin string,
	name string,
) bool {
	if collection == nil || len(collection.Freight) == 0 {
		return false
	}
	freightRef, ok := collection.Freight[origin]
	return ok && freightRef.Name == name
}

// handleDelete handles the deletion of the given Stage. It clears the
// verification status of all Freight that have been verified in the Stage, the
// approval status of all Freight that have been approved for the Stage, and
// deletes all AnalysisRuns that are associated with the Stage.
//
// It returns an error aggregate of all errors that occurred during the deletion
// process.
func (r *RegularStageReconciler) handleDelete(ctx context.Context, stage *kargoapi.Stage) error {
	return handleDelete(ctx, r.cfg, r.client, stage)
}

func handleDelete(ctx context.Context, cfg ReconcilerConfig, cl client.Client, stage *kargoapi.Stage) error {
	// If the Stage does not have the finalizer, there is nothing to do.
	if !controllerutil.ContainsFinalizer(stage, kargoapi.FinalizerName) {
		return nil
	}

	// Clear the verification and approval status of all Freight that have been
	// verified or approved for the Stage, and delete all AnalysisRuns.
	toClear := []func(context.Context, ReconcilerConfig, client.Client, *kargoapi.Stage) error{
		clearVerifications,
		clearApprovals,
		clearAnalysisRuns,
	}
	var errs []error
	for _, c := range toClear {
		if err := c(ctx, cfg, cl, stage); err != nil {
			errs = append(errs, err)
		}
	}
	if err := kerrors.Flatten(kerrors.NewAggregate(errs)); err != nil {
		// We ran into an error, return it before proceeding with removing the
		// finalizer.
		return fmt.Errorf("error handling deletion of Stage: %w", err)
	}

	// Remove the finalizer from the Stage.
	if err := api.RemoveFinalizer(ctx, cl, stage); err != nil {
		return fmt.Errorf("error removing finalizer from Stage: %w", err)
	}

	return nil
}

// clearVerifications clears the verification status of all Freight that have
// been verified in the given Stage. It removes the Stage from the VerifiedIn
// map of each Freight.
func clearVerifications(ctx context.Context, _ ReconcilerConfig, cl client.Client, stage *kargoapi.Stage) error {
	verified := kargoapi.FreightList{}
	if err := cl.List(
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

		if err := kubeclient.PatchStatus(ctx, cl, &f, func(status *kargoapi.FreightStatus) {
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
func clearApprovals(ctx context.Context, _ ReconcilerConfig, cl client.Client, stage *kargoapi.Stage) error {
	approved := kargoapi.FreightList{}
	if err := cl.List(
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

		if err := kubeclient.PatchStatus(ctx, cl, &f, func(status *kargoapi.FreightStatus) {
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
func clearAnalysisRuns(ctx context.Context, cfg ReconcilerConfig, cl client.Client, stage *kargoapi.Stage) error {
	if !cfg.RolloutsIntegrationEnabled {
		return nil
	}

	if err := cl.DeleteAllOf(
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
func summarizeConditions(stage *kargoapi.Stage, newStatus *kargoapi.StageStatus, err error) bool {
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
		return false
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
		return false
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
		return false
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
		return false
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
		return false
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
	return true
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
