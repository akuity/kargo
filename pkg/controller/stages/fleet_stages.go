package stages

// This reconciler is responsible for reconciling stages with `targets` specified.
// These stages will create PromotionRequests instead of Promotions.
// Some features of regular_stages are not supported for fleet stages:
// - Healthchecks
// - ArgoCD app references
// - AutoPromotion holds
// - Rollbacks
// - MatchUpstream autopromotions

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/controller"
	"github.com/akuity/kargo/pkg/controller/metrics"
	"github.com/akuity/kargo/pkg/credentials"
	kargoEvent "github.com/akuity/kargo/pkg/event"
	k8sevent "github.com/akuity/kargo/pkg/event/kubernetes"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/kargo"
	"github.com/akuity/kargo/pkg/kubeclient"
	libEvent "github.com/akuity/kargo/pkg/kubernetes/event"
	"github.com/akuity/kargo/pkg/logging"
	intpredicate "github.com/akuity/kargo/pkg/predicate"
	"github.com/akuity/kargo/pkg/telemetry"
)

type FleetStageReconciler struct {
	cfg            ReconcilerConfig
	client         client.Client
	credentialsDB  credentials.Database
	eventSender    kargoEvent.Sender
	shardPredicate controller.ResponsibleFor[kargoapi.Stage]

	backoffCfg wait.Backoff
}

// NewFleetStageReconciler creates a new Stages reconciler.
func NewFleetStageReconciler(
	cfg ReconcilerConfig,
	credentialsDB credentials.Database,
) *FleetStageReconciler {
	return &FleetStageReconciler{
		cfg:           cfg,
		credentialsDB: credentialsDB,
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
func (r *FleetStageReconciler) SetupWithManager(
	ctx context.Context,
	kargoMgr ctrl.Manager,
	_ ctrl.Manager,
	sharedIndexer client.FieldIndexer,
) error {
	// Configure client and event recorder using manager.
	r.client = kargoMgr.GetClient()
	r.eventSender = k8sevent.NewEventSender(
		libEvent.NewRecorder(ctx, kargoMgr.GetScheme(), kargoMgr.GetClient(), r.cfg.Name()),
	)

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
		Named("fleet_stage").
		WithEventFilter(controller.ResponsibleFor[client.Object]{
			IsDefaultController: r.cfg.IsDefaultController,
			ShardName:           r.cfg.ShardName,
		}).
		WithEventFilter(intpredicate.IgnoreDelete[client.Object]{}).
		WithEventFilter(
			predicate.And(
				IsTargetAwareStage(true),
				IsControlFlowStage(false),
				predicate.Or(
					predicate.GenerationChangedPredicate{},
					kargo.RefreshRequested{},
					kargo.ReverifyRequested{},
					kargo.VerificationAbortRequested{},
				),
			),
		).
		WithOptions(controller.CommonOptions(r.cfg.MaxConcurrentFleetReconciles)).
		Build(r)
	if err != nil {
		return fmt.Errorf("error building Stage reconciler: %w", err)
	}

	// Configure the watches.
	// Changes to these objects that match the constraints from the predicates
	// will enqueue a reconciliation for the related Stage(s).
	logger := logging.LoggerFromContext(ctx)

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
		"maxConcurrentReconciles", r.cfg.MaxConcurrentFleetReconciles,
	)

	return nil
}

func (r *FleetStageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// NOTE: we share tracer context between all stage reconcilers
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

	// Safety check: do not reconcile Stages that are not fleet Stages.
	if !stage.IsTargetAware() {
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

func (r *FleetStageReconciler) reconcile(
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
		if summarizeFleetConditions(working, &newStatus, err) {
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
func (r *FleetStageReconciler) syncPromotionRequests(
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
		newStatus.CurrentPromotionRequest = &kargoapi.PromotionRequestReference{
			Name: highestPrioRequest.Name,
		}
		if highestPrioRequest.Status.Freight != nil {
			newStatus.CurrentPromotionRequest.Freight = highestPrioRequest.Status.Freight.DeepCopy()
		}
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

	// Process all unprocessed terminal promotion requests
	for _, promotionRequest := range newRequests {
		err := r.recordTerminalRequest(&newStatus, promotionRequest, stage.Generation)
		if err != nil {
			return newStatus, err
		}
	}

	return newStatus, nil
}

// recordTerminalRequest records terminal promotion request in the Stage status
// It updates LastPromotionRequest,
// and for successful requests updates FreightHistory and resets health and verification
func (r *FleetStageReconciler) recordTerminalRequest(
	newStatus *kargoapi.StageStatus,
	promotionRequest *kargoapi.PromotionRequest,
	generation int64,
) error {
	newStatus.LastPromotionRequest = newLastPromotionRequestReference(promotionRequest)
	if promotionRequest.Status.Phase == kargoapi.PromotionRequestPhaseSucceeded {
		// All successful promotion requests should have a freight collection
		if promotionRequest.Status.FreightCollection == nil {
			return fmt.Errorf("invalid PromotionRequest status: expected FreightCollection")
		}
		newStatus.FreightHistory.Record(promotionRequest.Status.FreightCollection)
		// The Stage is running new Freight: what was known about the health and
		// verification of the old is no longer relevant.
		newStatus.Health = nil
		conditions.Set(newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionUnknown,
			Reason:             "WaitingForHealthCheck",
			Message:            "Waiting for health check to be performed after successful promotion",
			ObservedGeneration: generation,
		})
		conditions.Set(newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeVerified,
			Status:             metav1.ConditionUnknown,
			Reason:             "WaitingForVerification",
			Message:            "Waiting for verification to be performed after successful promotion",
			ObservedGeneration: generation,
		})
	}
	return nil
}

// newLastPromotionRequestReference builds the reference a Stage records for one of
// its PromotionRequests. The reference names the PromotionRequest's Freight
// rather than describing it; a reader that needs the Freight's contents can
// look them up from the Freight itself.
func newLastPromotionRequestReference(
	promotionRequest *kargoapi.PromotionRequest,
) *kargoapi.PromotionRequestReference {
	var freightRef *kargoapi.FreightReference
	if promotionRequest.Status.Freight != nil {
		freightRef = promotionRequest.Status.Freight.DeepCopy()
	}

	return &kargoapi.PromotionRequestReference{
		Name:       promotionRequest.Name,
		Phase:      promotionRequest.Status.Phase,
		Message:    promotionRequest.Status.Message,
		FinishedAt: promotionRequest.Status.FinishedAt,
		Freight:    freightRef,
		// We don't inherit freight collection here, instead it's done in the PromotionRequest reconciler
		FreightCollection: promotionRequest.Status.FreightCollection,
	}
}

// assessHealth assesses the health of a Stage based on the health checks from
// the last Promotion.
func (r *FleetStageReconciler) assessHealth(ctx context.Context, stage *kargoapi.Stage) kargoapi.StageStatus {
	logger := logging.LoggerFromContext(ctx)
	newStatus := *stage.Status.DeepCopy()

	if stage.Status.CurrentPromotionRequest != nil {
		logger.Debug("PromotionRequest is in progress: no health checks to perform")
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionUnknown,
			Reason:             "ActivePromotion",
			Message:            "Stage has a PromotionRequest in progress",
			ObservedGeneration: stage.Generation,
		})
		newStatus.Health = &kargoapi.Health{
			Status: kargoapi.HealthStateUnknown,
			Issues: []string{
				"Cannot assess health because a PromotionRequest is currently in progress",
			},
		}
		return newStatus
	}
	lastPromo := stage.Status.LastPromotionRequest
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
	if lastPromo.Phase != kargoapi.PromotionRequestPhaseSucceeded {
		logger.Debug("Last promotion did not succeed: defaulting Stage health to Unknown")
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionUnknown,
			Reason:             fmt.Sprintf("LastPromotionRequest%s", lastPromo.Phase),
			Message:            "Cannot assess health because last PromotionRequest did not succeed",
			ObservedGeneration: stage.Generation,
		})
		newStatus.Health = &kargoapi.Health{
			Status: kargoapi.HealthStateUnknown,
			Issues: []string{"Cannot assess health because last PromotionRequest did not succeed"},
		}
		return newStatus
	}

	// HealthChecks are not supported yet for fleet stages.
	// Health condition status is always healthy for now
	logger.Debug("Stage promotes to Targets: no Stage-level health checks to perform")
	conditions.Set(&newStatus, &metav1.Condition{
		Type:               kargoapi.ConditionTypeHealthy,
		Status:             metav1.ConditionTrue,
		Reason:             "TargetAwareStage",
		Message:            "Health is assessed per Target, not for the Stage",
		ObservedGeneration: stage.Generation,
	})
	// TODO: we currently set status to Healthy to support verifications.
	// We need to change verification logic to support HealthStateNotApplicable and use it here
	newStatus.Health = &kargoapi.Health{
		Status: kargoapi.HealthStateHealthy,
	}
	return newStatus
}

// syncFreight ensures that all Freight statuses accurately reflect whether they
// are currently in use by the Stage.
func (r *FleetStageReconciler) syncFreight(ctx context.Context, stage *kargoapi.Stage) error {
	// syncFreight is shared with regular stages reconciler
	return syncFreight(ctx, r.client, stage)
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
func (r *FleetStageReconciler) verifyStageFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
	startTime time.Time,
	endTime func() time.Time,
) (newStatus kargoapi.StageStatus, err error) {
	// Verification logic is shared with regular stages
	ver := verifier{
		cfg:           r.cfg,
		client:        r.client,
		credentialsDB: r.credentialsDB,
		eventSender:   r.eventSender,
		backoffCfg:    r.backoffCfg,
	}
	return ver.verifyStageFreight(ctx, stage, startTime, endTime)
}

// markFreightVerifiedForStage marks the Freight that is associated with the
// Stage as verified. If the Freight has already been verified, then no action
// is taken.
func (r *FleetStageReconciler) markFreightVerifiedForStage(
	ctx context.Context,
	stage *kargoapi.Stage,
) (kargoapi.StageStatus, error) {
	// markFreightVerifiedForStage is shared with regular stages
	return markFreightVerifiedForStage(ctx, r.client, stage)
}

// autoPromoteFreight automatically promotes the candidate Freight for each
// requested origin, unless auto-promotion is disabled or the origin has an
// effective auto-promotion hold.
func (r *FleetStageReconciler) autoPromoteFreight(
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
		// Nothing to promote.
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

		candidate, exists := candidates[origin]
		if !exists {
			logger.Debug("no Freight from origin available for auto-promotion", "origin", origin)
			continue
		}

		autoPromote, err := r.shouldAutoPromoteRequestedFreight(ctx, origin, candidate, currentFreight, stage)
		if err != nil {
			return newStatus, err
		}

		if autoPromote {
			if err = r.createAutoPromotionRequest(ctx, stage, &candidate, origin); err != nil {
				return newStatus, err
			}
		}
	}

	return newStatus, nil
}

func (r *FleetStageReconciler) shouldAutoPromoteRequestedFreight(
	ctx context.Context,
	origin string,
	candidate kargoapi.Freight,
	currentFreight *kargoapi.FreightCollection,
	stage *kargoapi.Stage) (bool, error) {

	// TODO: implement autopromotion holds for PromotionRequests: do not create promotion if hold is in place
	freightLogger := logging.LoggerFromContext(ctx).WithValues("origin", origin, "freight", candidate.Name)

	// Only proceed if the candidate Freight is not already current in the Stage.
	if freightCollectionHasFreight(currentFreight, origin, candidate.Name) {
		freightLogger.Debug("Stage already has candidate Freight for origin")
		return false, nil
	}
	if stageAwaitingFreightForOrigin(stage, origin, candidate.Name) {
		freightLogger.Debug("Stage is already awaiting candidate Freight for origin")
		return false, nil
	}

	exists, err := r.promotionRequestExistsForStageFreight(ctx, stage, candidate.Name)
	if err != nil {
		return false, fmt.Errorf(
			"error listing existing PromotionRequests for Freight %q in namespace %q: %w",
			candidate.Name, stage.Namespace, err,
		)
	}
	if exists {
		freightLogger.Debug("a PromotionRequest already exists for Stage and Freight")
		return false, nil
	}

	return true, nil
}

// createAutoPromotionRequest creates a PromotionRequest expressing the intent
// to promote the candidate Freight to the Targets that the target-aware Stage
// governs. Those Targets are resolved once, as the request is built, and
// recorded on it; the request does not promote anything itself.
//
// The guard against duplicate work here is deliberately stricter than the one
// autoPromoteFreight applies to Promotions: a PromotionRequest is created only when
// no PromotionRequest for this Stage and Freight exists at all, in any phase.
// Stage status now records the current and last PromotionRequest, but those are
// mirrors of a request's own phase, not of a Stage having absorbed its outcome:
// a PromotionRequest promotes nothing itself, so there is still no equivalent of
// "succeeded, but the outcome is not yet recorded in status" to reason about --
// and absent a guard that holds unconditionally, every reconcile would create
// another PromotionRequest.
//
// The guard is confined to auto-promotion. Promoting the same Freight to the
// same Stage again deliberately -- rolling back to it, say -- goes through the
// API server, which creates a PromotionRequest unconditionally.
func (r *FleetStageReconciler) createAutoPromotionRequest(
	ctx context.Context,
	stage *kargoapi.Stage,
	candidate *kargoapi.Freight,
	origin string,
) error {
	logger := logging.LoggerFromContext(ctx).WithValues(
		"origin", origin,
		"freight", candidate.Name,
	)

	promotionRequest, err := api.NewPromotionRequest(ctx, r.client, stage, candidate.Name)
	if err != nil {
		return fmt.Errorf(
			"error building PromotionRequest for Freight %q in namespace %q: %w",
			candidate.Name, stage.Namespace, err,
		)
	}

	if err = r.client.Create(ctx, promotionRequest); err != nil {
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
			logger.Debug(
				"auto-promotion was denied by an admission webhook",
				"error", err.Error(),
			)
			return nil
		}
		return fmt.Errorf(
			"error creating PromotionRequest for Freight %q in namespace %q: %w",
			candidate.Name, stage.Namespace, err,
		)
	}

	// FIXME: send promotion request events
	logger.Debug(
		"created PromotionRequest resource",
		"promotionRequest", promotionRequest.Name,
	)
	return nil
}

// promotionRequestExistsForStageFreight reports whether any PromotionRequest exists for
// the given Stage and Freight, in any phase.
func (r *FleetStageReconciler) promotionRequestExistsForStageFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
	freightName string,
) (bool, error) {
	promotionRequests := &kargoapi.PromotionRequestList{}
	if err := r.client.List(
		ctx,
		promotionRequests,
		client.InNamespace(stage.Namespace),
		client.MatchingFieldsSelector{
			Selector: fields.OneTermEqualSelector(
				indexer.PromotionRequestsByStageAndFreightField,
				indexer.StageAndFreightKey(stage.Name, freightName),
			),
		},
	); err != nil {
		return false, err
	}
	return len(promotionRequests.Items) > 0, nil
}

// handleDelete handles the deletion of the given Stage. It clears the
// verification status of all Freight that have been verified in the Stage, the
// approval status of all Freight that have been approved for the Stage, and
// deletes all AnalysisRuns that are associated with the Stage.
//
// It returns an error aggregate of all errors that occurred during the deletion
// process.
func (r *FleetStageReconciler) handleDelete(ctx context.Context, stage *kargoapi.Stage) error {
	return handleDelete(ctx, r.cfg, r.client, stage)
}

// summarizeFleetConditions summarizes the conditions of the given Stage.
// It extends summarizeConditions by checking last promotion request status.
// See summarizeConditions.
func summarizeFleetConditions(stage *kargoapi.Stage, newStatus *kargoapi.StageStatus, err error) bool {
	baseReady := summarizeConditions(stage, newStatus, err)
	if err != nil {
		return baseReady
	}
	promoCond := conditions.Get(newStatus, kargoapi.ConditionTypePromoting)
	if promoCond != nil {
		return baseReady
	}

	// If we are not currently Promoting but the last promotion failed,
	// then we are not Ready.
	if lastPromoRequest := newStatus.LastPromotionRequest; lastPromoRequest != nil &&
		lastPromoRequest.Phase.IsTerminal() &&
		lastPromoRequest.Phase != kargoapi.PromotionRequestPhaseSucceeded {
		conditions.Set(newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             fmt.Sprintf("LastPromotionRequest%s", string(lastPromoRequest.Phase)),
			Message:            lastPromoRequest.Message,
			ObservedGeneration: stage.Generation,
		})
		return false
	}

	return baseReady
}
