package stages

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/conditions"
	"github.com/akuity/kargo/internal/controller"
	argocdapi "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	rolloutsapi "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
	"github.com/akuity/kargo/internal/directives"
	kargoEvent "github.com/akuity/kargo/internal/event"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/kargo"
	"github.com/akuity/kargo/internal/kubeclient"
	libEvent "github.com/akuity/kargo/internal/kubernetes/event"
	"github.com/akuity/kargo/internal/logging"
	intpredicate "github.com/akuity/kargo/internal/predicate"
	"github.com/akuity/kargo/internal/rollouts"
)

// ReconcilerConfig represents configuration for the stage reconciler.
type ReconcilerConfig struct {
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
	cfg              ReconcilerConfig
	client           client.Client
	eventRecorder    record.EventRecorder
	directivesEngine directives.Engine

	backoffCfg wait.Backoff
}

// NewRegularStageReconciler creates a new Stages reconciler.
func NewRegularStageReconciler(cfg ReconcilerConfig, engine directives.Engine) *RegularStageReconciler {
	return &RegularStageReconciler{
		cfg:              cfg,
		directivesEngine: engine,
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
	r.eventRecorder = libEvent.NewRecorder(ctx, kargoMgr.GetScheme(), kargoMgr.GetClient(), r.cfg.Name())

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

	// Build the controller with the reconciler.
	c, err := ctrl.NewControllerManagedBy(kargoMgr).
		For(&kargoapi.Stage{}).
		WithOptions(controller.CommonOptions(r.cfg.MaxConcurrentReconciles)).
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

	// Watch for Freight that has been marked as verified in a Stage and enqueue
	// downstream Stages for reconciliation.
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
			indexer.StagesByAnalysisRun(r.cfg.RolloutsControllerInstanceID),
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
		"namespace", req.NamespacedName.Namespace,
		"stage", req.NamespacedName.Name,
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
	newStatus, needsRequeue, reconcileErr := r.reconcile(ctx, stage, time.Now())
	logger.Debug("done reconciling Stage")

	// Record the current refresh token as having been handled.
	if token, ok := kargoapi.RefreshAnnotationValue(stage.GetAnnotations()); ok {
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
		return ctrl.Result{Requeue: true}, nil
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
	newStatus := *stage.Status.DeepCopy()

	// Mark the Stage as reconciling.
	conditions.Set(&newStatus, &metav1.Condition{
		Type:               kargoapi.ConditionTypeReconciling,
		Status:             metav1.ConditionTrue,
		Reason:             "Reconciling",
		ObservedGeneration: stage.Generation,
	})

	var requestRequeue bool
	subReconcilers := []struct {
		name      string
		reconcile func() (kargoapi.StageStatus, error)
	}{
		{
			name: "syncing Promotions",
			reconcile: func() (kargoapi.StageStatus, error) {
				status, hasPendingPromotions, err := r.syncPromotions(ctx, stage)
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
			name: "assessing health",
			reconcile: func() (kargoapi.StageStatus, error) {
				return r.assessHealth(ctx, stage), nil
			},
		},
		{
			name: "verifying Stage Freight",
			reconcile: func() (kargoapi.StageStatus, error) {
				status, err := r.verifyStageFreight(ctx, stage, startTime, time.Now)
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
				status, err := r.markFreightVerifiedForStage(ctx, stage)
				if err != nil {
					err = fmt.Errorf("failed to verify Freight for Stage: %w", err)
				}
				return status, err
			},
		},
		{
			name: "auto-promoting Freight",
			reconcile: func() (kargoapi.StageStatus, error) {
				status, err := r.autoPromoteFreight(ctx, stage)
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
		summarizeConditions(stage, &newStatus, err)

		// If an error occurred during the sub-reconciler, then we should
		// return the error which will cause the Stage to be requeued.
		if err != nil {
			return newStatus, false, err
		}

		// Patch the status of the Stage after each sub-reconciler to show
		// progress.
		if err = kubeclient.PatchStatus(ctx, r.client, stage, func(status *kargoapi.StageStatus) {
			*status = newStatus
		}); err != nil {
			logger.Error(err, fmt.Sprintf("failed to update Stage status after %s", subR.name))
		}
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
	slices.SortFunc(promotions.Items, kargoapi.ComparePromotionByPhaseAndCreationTime)

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
		// Gather information about the Promotions that have terminated since the
		// last reconciliation.
		var newPromotions []kargoapi.PromotionReference
		for _, promo := range promotions.Items {
			// Update the conditions to reflect that we are no longer promoting.
			conditions.Delete(&newStatus, kargoapi.ConditionTypePromoting)
			newStatus.CurrentPromotion = nil

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
				info := kargoapi.PromotionReference{
					Name:       promo.Name,
					Status:     promo.Status.DeepCopy(),
					FinishedAt: promo.Status.FinishedAt,
				}
				if promo.Status.Freight != nil {
					info.Freight = promo.Status.Freight.DeepCopy()
				}
				newPromotions = append(newPromotions, info)
			}
		}

		// As we will be appending to the Freight history, we need to ensure that
		// we order the Promotions from oldest to newest. This is because the
		// Freight history is garbage collected based on the number of entries,
		// and we want to ensure that the oldest entries are removed first.
		slices.SortFunc(newPromotions, func(a, b kargoapi.PromotionReference) int {
			return strings.Compare(a.Name, b.Name)
		})

		// Update the Stage status with the information about the newly terminated
		// Promotions, and any new Freight that was successfully promoted.
		for _, p := range newPromotions {
			promo := p
			newStatus.LastPromotion = &promo
			if p.Status.Phase == kargoapi.PromotionPhaseSucceeded {
				// If the Promotion was successful, then we should add the Freight
				// to the history of successfully promoted Freight.
				newStatus.FreightHistory.Record(promo.Status.FreightCollection)

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
		if stage.Status.Health == nil || stage.Status.Health.Status != kargoapi.HealthStateUnhealthy {
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
	//
	// TODO(hidde): Long term, this should probably be changed to allow to
	//  continue to run health checks from the last successful Promotion,
	//  even if the current Promotion did not succeed (e.g. because it was
	//  aborted).
	if lastPromo.Status.Phase != kargoapi.PromotionPhaseSucceeded {
		logger.Debug("Last promotion did not succeed: defaulting Stage health to Unhealthy")
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionFalse,
			Reason:             fmt.Sprintf("LastPromotion%s", lastPromo.Status.Phase),
			Message:            "Last Promotion did not succeed",
			ObservedGeneration: stage.Generation,
		})
		newStatus.Health = &kargoapi.Health{
			Status: kargoapi.HealthStateUnhealthy,
			Issues: []string{"Last Promotion did not succeed"},
		}
		return newStatus
	}

	// Compose the health check steps.
	healthChecks := lastPromo.Status.HealthChecks
	var steps []directives.HealthCheckStep
	for _, step := range healthChecks {
		steps = append(steps, directives.HealthCheckStep{
			Kind:   step.Uses,
			Config: step.GetConfig(),
		})
	}

	// Run the health checks.
	health := r.directivesEngine.CheckHealth(ctx, directives.HealthCheckContext{
		Project: stage.Namespace,
		Stage:   stage.Name,
	}, steps)
	newStatus.Health = &health

	// Set the Healthy condition based on the health status.
	switch health.Status {
	case kargoapi.HealthStateHealthy:
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionTrue,
			Reason:             string(health.Status),
			Message:            fmt.Sprintf("Stage is healthy (performed %d health checks)", len(healthChecks)),
			ObservedGeneration: stage.Generation,
		})
	case kargoapi.HealthStateUnhealthy:
		conditions.Set(&newStatus, &metav1.Condition{
			Type:   kargoapi.ConditionTypeHealthy,
			Status: metav1.ConditionFalse,
			Reason: string(health.Status),
			Message: fmt.Sprintf(
				"Stage is unhealthy (%d issues in %d health checks)",
				len(health.Issues), len(healthChecks),
			),
			ObservedGeneration: stage.Generation,
		})
	default:
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeHealthy,
			Status:             metav1.ConditionUnknown,
			Reason:             string(health.Status),
			ObservedGeneration: stage.Generation,
		})
	}

	return newStatus
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
	logger := logging.LoggerFromContext(ctx)
	newStatus = *stage.Status.DeepCopy()

	// If there is no current Freight, then we have nothing to verify.
	curFreight := stage.Status.FreightHistory.Current()
	if curFreight == nil {
		logger.Debug("Stage has no current Freight: no verification to perform")
		conditions.Set(&newStatus, &metav1.Condition{
			Type:               kargoapi.ConditionTypeVerified,
			Status:             metav1.ConditionUnknown,
			Reason:             "NoFreight",
			Message:            "Stage has no current Freight to verify",
			ObservedGeneration: stage.Generation,
		})
		return newStatus, nil
	}

	// If we are currently promoting Freight, then we are not in a stable state
	// and should wait until the promotion is complete.
	if curPromotion := stage.Status.CurrentPromotion; curPromotion != nil {
		logger.Debug("Stage is currently promoting Freight: skipping verification")
		return newStatus, nil
	}

	defer func() {
		curFreight = newStatus.FreightHistory.Current()
		if curFreight == nil || len(curFreight.VerificationHistory) == 0 {
			return
		}

		for _, vi := range curFreight.VerificationHistory {
			if vi.Phase == kargoapi.VerificationPhaseSuccessful {
				// If the Freight has at least one successful verification,
				// then we can consider the Freight to be verified.
				conditions.Set(&newStatus, &metav1.Condition{
					Type:               kargoapi.ConditionTypeVerified,
					Status:             metav1.ConditionTrue,
					Reason:             "Verified",
					Message:            "Freight has been verified",
					ObservedGeneration: stage.Generation,
				})
				return
			}
		}

		// If the Freight has no successful verification, then we should look
		// for the most recent verification and set the status accordingly.
		lastVerification := curFreight.VerificationHistory.Current()
		if lastVerification != nil {
			switch lastVerification.Phase {
			case kargoapi.VerificationPhasePending:
				conditions.Set(&newStatus, &metav1.Condition{
					Type:               kargoapi.ConditionTypeVerified,
					Status:             metav1.ConditionUnknown,
					Reason:             "VerificationPending",
					Message:            "Freight is pending verification",
					ObservedGeneration: stage.Generation,
				})
			case kargoapi.VerificationPhaseRunning:
				conditions.Set(&newStatus, &metav1.Condition{
					Type:               kargoapi.ConditionTypeVerified,
					Status:             metav1.ConditionUnknown,
					Reason:             "VerificationRunning",
					Message:            "Freight is currently being verified",
					ObservedGeneration: stage.Generation,
				})
			case kargoapi.VerificationPhaseFailed, kargoapi.VerificationPhaseError:
				conditions.Set(&newStatus, &metav1.Condition{
					Type:               kargoapi.ConditionTypeVerified,
					Status:             metav1.ConditionFalse,
					Reason:             fmt.Sprintf("Verification%s", lastVerification.Phase),
					Message:            lastVerification.Message,
					ObservedGeneration: stage.Generation,
				})
			case kargoapi.VerificationPhaseAborted:
				conditions.Set(&newStatus, &metav1.Condition{
					Type:               kargoapi.ConditionTypeVerified,
					Status:             metav1.ConditionFalse,
					Reason:             "VerificationAborted",
					Message:            lastVerification.Message,
					ObservedGeneration: stage.Generation,
				})
			case kargoapi.VerificationPhaseInconclusive:
				conditions.Set(&newStatus, &metav1.Condition{
					Type:               kargoapi.ConditionTypeVerified,
					Status:             metav1.ConditionUnknown,
					Reason:             "VerificationInconclusive",
					Message:            lastVerification.Message,
					ObservedGeneration: stage.Generation,
				})
			default:
				conditions.Set(&newStatus, &metav1.Condition{
					Type:    kargoapi.ConditionTypeVerified,
					Status:  metav1.ConditionUnknown,
					Reason:  "UnknownVerificationPhase",
					Message: fmt.Sprintf("Freight verification is in an unknown phase: %s", lastVerification.Phase),
				})
			}
		}
	}()

	// Get the re-verification request, if any.
	reverifyReq, _ := kargoapi.ReverifyAnnotationValue(stage.GetAnnotations())

	// Check if the current Freight has already been verified.
	var newVI *kargoapi.VerificationInfo
	if lastVerification := curFreight.VerificationHistory.Current(); lastVerification != nil {
		// If the last verification is not terminal, then we should check if
		// we need to abort the verification, or if we need to get the verification
		// result.
		if !lastVerification.Phase.IsTerminal() {
			// Check if we need to abort the verification.
			abortReq, _ := kargoapi.AbortVerificationAnnotationValue(stage.GetAnnotations())
			if abortReq.ForID(lastVerification.ID) {
				logger.Debug("aborting verification of Stage Freight")

				// Abort the verification.
				newVI, err = r.abortVerification(ctx, *curFreight, abortReq)
				if newVI != nil {
					newStatus.FreightHistory.Current().VerificationHistory.UpdateOrPush(*newVI)
				}

				// Issue an event for the aborted verification.
				for _, ref := range curFreight.Freight {
					r.recordFreightVerificationEvent(stage, ref, newVI)
				}

				return newStatus, err
			}

			// Get the latest result of the verification.
			newVI, err = r.getVerificationResult(ctx, *curFreight)
			if newVI != nil {
				newStatus.FreightHistory.Current().VerificationHistory.UpdateOrPush(*newVI)

				// If the verification is terminal, we should issue an event for
				// each Freight that was verified.
				if newVI.Phase.IsTerminal() {
					for _, ref := range curFreight.Freight {
						r.recordFreightVerificationEvent(stage, ref, newVI)
					}
				}
			}
			return newStatus, err
		}

		// If the last verification is terminal, and we are not re-verifying
		// the Freight, then we have nothing to do.
		if !reverifyReq.ForID(lastVerification.ID) {
			logger.Debug("Stage Freight has already been verified")
			return newStatus, nil
		}
	}

	// If the Stage is not passed any health checks (yet), then we should not
	// verify the Freight.
	if stage.Status.Health == nil || stage.Status.Health.Status != kargoapi.HealthStateHealthy {
		logger.Debug("Stage has not passed health checks: skipping verification")
		return newStatus, nil
	}

	// If we have no specific verification configuration, then we can mark the
	// verification as successful.
	if stage.Spec.Verification == nil {
		newVI := kargoapi.VerificationInfo{
			StartTime:  ptr.To(metav1.NewTime(startTime)),
			FinishTime: ptr.To(metav1.NewTime(endTime())),
			Phase:      kargoapi.VerificationPhaseSuccessful,
		}
		newStatus.FreightHistory.Current().VerificationHistory.UpdateOrPush(newVI)

		// Issue an event for each Freight that was verified.
		for _, ref := range curFreight.Freight {
			r.recordFreightVerificationEvent(stage, ref, &newVI)
		}
		return newStatus, nil
	}

	// Start a new (re-)verification.
	newVI, err = r.startVerification(ctx, stage, *curFreight, reverifyReq, startTime)
	if newVI != nil {
		newStatus.FreightHistory.Current().VerificationHistory.UpdateOrPush(*newVI)

		// There is a chance for the verification to be terminal immediately
		// after starting it. For example, if the rollouts integration is not
		// enabled. In this case, we should issue an event for the verification.
		if newVI.Phase.IsTerminal() {
			for _, ref := range curFreight.Freight {
				r.recordFreightVerificationEvent(stage, ref, newVI)
			}
		}
	}
	return newStatus, err
}

// markFreightVerifiedForStage marks the Freight that is associated with the
// Stage as verified. If the Freight has already been verified, then no action
// is taken.
func (r *RegularStageReconciler) markFreightVerifiedForStage(
	ctx context.Context,
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
		if err := r.client.Get(ctx, types.NamespacedName{
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
		if _, ok := freight.Status.VerifiedIn[stage.Name]; ok {
			logger.Debug("Freight has already been verified in Stage")
			continue
		}

		// Verify the Freight.
		if err := kubeclient.PatchStatus(ctx, r.client, freight, func(status *kargoapi.FreightStatus) {
			if status.VerifiedIn == nil {
				status.VerifiedIn = make(map[string]kargoapi.VerifiedStage)
			}
			status.VerifiedIn[stage.Name] = kargoapi.VerifiedStage{
				VerifiedAt: curFreight.VerificationHistory.Current().FinishTime.DeepCopy(),
			}
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

// recordFreightVerificationEvent records an event for the verification of a
// Freight. The event contains information about the Freight, the verification,
// and the Stage that triggered the verification.
func (r *RegularStageReconciler) recordFreightVerificationEvent(
	stage *kargoapi.Stage,
	freightRef kargoapi.FreightReference,
	vi *kargoapi.VerificationInfo,
) {
	freight := &kargoapi.Freight{}
	if err := r.client.Get(context.Background(), types.NamespacedName{
		Namespace: stage.Namespace,
		Name:      freightRef.Name,
	}, freight); err != nil {
		logging.LoggerFromContext(context.Background()).Error(
			err, "failed to get Freight for verification event",
			"freight", freightRef.Name,
		)
		return
	}

	annotations := map[string]string{
		kargoapi.AnnotationKeyEventActor:             kargoapi.FormatEventControllerActor(r.cfg.Name()),
		kargoapi.AnnotationKeyEventProject:           stage.Namespace,
		kargoapi.AnnotationKeyEventStageName:         stage.Name,
		kargoapi.AnnotationKeyEventFreightAlias:      freight.Alias,
		kargoapi.AnnotationKeyEventFreightName:       freight.Name,
		kargoapi.AnnotationKeyEventFreightCreateTime: freight.CreationTimestamp.Format(time.RFC3339),
	}
	if vi.StartTime != nil {
		annotations[kargoapi.AnnotationKeyEventVerificationStartTime] = vi.StartTime.Format(time.RFC3339)
	}
	if vi.FinishTime != nil {
		annotations[kargoapi.AnnotationKeyEventVerificationFinishTime] = vi.FinishTime.Format(time.RFC3339)
	}

	// Extract metadata from the AnalysisRun if available
	if vi.HasAnalysisRun() {
		annotations[kargoapi.AnnotationKeyEventAnalysisRunName] = vi.AnalysisRun.Name

		ar := &rolloutsapi.AnalysisRun{}
		if err := r.client.Get(context.Background(), types.NamespacedName{
			Namespace: vi.AnalysisRun.Namespace,
			Name:      vi.AnalysisRun.Name,
		}, ar); err != nil {
			// Log the error but do not fail the event recording.
			logging.LoggerFromContext(context.Background()).Error(
				err, "failed to get AnalysisRun for verification event",
				"analysisRun", vi.AnalysisRun.Name, "freight", freightRef.Name,
			)
		}
		// AnalysisRun that triggered by a Promotion contains the Promotion name
		if promoName, ok := ar.Labels[kargoapi.PromotionLabelKey]; ok {
			annotations[kargoapi.AnnotationKeyEventPromotionName] = promoName
		}
	}

	// If the verification is manually triggered (e.g. reverify),
	// override the actor with the one who triggered the verification.
	if vi.Actor != "" {
		annotations[kargoapi.AnnotationKeyEventActor] = vi.Actor
	}

	reason := kargoapi.EventReasonFreightVerificationUnknown
	message := vi.Message

	switch vi.Phase {
	case kargoapi.VerificationPhaseSuccessful:
		reason = kargoapi.EventReasonFreightVerificationSucceeded
		message = "Freight verification succeeded"
	case kargoapi.VerificationPhaseFailed:
		reason = kargoapi.EventReasonFreightVerificationFailed
	case kargoapi.VerificationPhaseError:
		reason = kargoapi.EventReasonFreightVerificationErrored
	case kargoapi.VerificationPhaseAborted:
		reason = kargoapi.EventReasonFreightVerificationAborted
	case kargoapi.VerificationPhaseInconclusive:
		reason = kargoapi.EventReasonFreightVerificationInconclusive
	}

	r.eventRecorder.AnnotatedEventf(freight, annotations, corev1.EventTypeNormal, reason, message)
}

// startVerification starts a new verification for the Freight that is associated
// with the Stage. If the Freight has already been verified, then no verification
// is started unless a re-verification is requested.
//
// If there is no verification configuration for the Stage, then the verification
// is automatically considered successful and no verification is started.
//
// If the Rollouts integration is disabled, then the verification is marked as
// failed with an appropriate message.
//
// To start a verification, the Stage must be healthy.
func (r *RegularStageReconciler) startVerification(
	ctx context.Context,
	stage *kargoapi.Stage,
	freight kargoapi.FreightCollection,
	req *kargoapi.VerificationRequest,
	startTime time.Time,
) (*kargoapi.VerificationInfo, error) {
	newVI := &kargoapi.VerificationInfo{
		ID:        uuid.NewString(),
		StartTime: &metav1.Time{Time: startTime},
	}

	// If we have a verification request, we should enrich the information
	// with the actor who requested the verification.
	curVI := freight.VerificationHistory.Current()
	if curVI != nil && req.ForID(curVI.ID) {
		newVI.Actor = req.Actor
	}

	// Return early, as we cannot start the verification if the Rollouts
	// integration is disabled.
	if !r.cfg.RolloutsIntegrationEnabled {
		newVI.FinishTime = ptr.To(metav1.Now())
		newVI.Phase = kargoapi.VerificationPhaseError
		newVI.Message = "Rollouts integration is disabled on this controller: cannot start verification"
		return newVI, nil
	}

	logger := logging.LoggerFromContext(ctx)

	// If this is not a re-verification request, check if there is an existing
	// AnalysisRun for the Stage and Freight. If there is, return the status
	// of the existing AnalysisRun.
	if req == nil {
		existingAnalysisRun, err := r.findExistingAnalysisRun(ctx, types.NamespacedName{
			Namespace: stage.Namespace,
			Name:      stage.Name,
		}, freight.ID)
		if err != nil {
			newVI.FinishTime = ptr.To(metav1.Now())
			newVI.Phase = kargoapi.VerificationPhaseError
			newVI.Message = err.Error()
			return newVI, nil
		}

		if existingAnalysisRun != nil {
			logger.Debug("AnalysisRun already exists for FreightCollection")

			newVI.FinishTime = existingAnalysisRun.Status.CompletedAt()
			newVI.Phase = kargoapi.VerificationPhase(existingAnalysisRun.Status.Phase)
			newVI.AnalysisRun = &kargoapi.AnalysisRunReference{
				Name:      existingAnalysisRun.Name,
				Namespace: existingAnalysisRun.Namespace,
				Phase:     string(existingAnalysisRun.Status.Phase),
			}
			newVI.FinishTime = existingAnalysisRun.Status.CompletedAt()
			return newVI, nil
		}
	}

	// At this point, we know that we need to start a new AnalysisRun for the
	// verification.
	builder := rollouts.NewAnalysisRunBuilder(r.client, rollouts.Config{
		ControllerInstanceID: r.cfg.RolloutsControllerInstanceID,
	})
	builderOpts := []rollouts.AnalysisRunOption{
		rollouts.WithNamePrefix(stage.Name),
		rollouts.WithNameSuffix(freight.ID),
		rollouts.WithExtraLabels(map[string]string{
			kargoapi.StageLabelKey:             stage.Name,
			kargoapi.FreightCollectionLabelKey: freight.ID,
		}),
	}
	for _, freightRef := range freight.Freight {
		builderOpts = append(builderOpts, rollouts.WithOwner{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Freight",
			Reference:  types.NamespacedName{Namespace: stage.Namespace, Name: freightRef.Name},
		})
	}
	if curVI == nil || (req.ForID(curVI.ID) && req.ControlPlane && req.Actor != "") {
		if stage.Status.LastPromotion != nil {
			builderOpts = append(builderOpts, rollouts.WithExtraLabels{
				kargoapi.PromotionLabelKey: stage.Status.LastPromotion.Name,
			})
		}
	}
	ar, err := builder.Build(ctx, stage.Namespace, stage.Spec.Verification, builderOpts...)
	if err != nil {
		newVI.FinishTime = ptr.To(metav1.Now())
		newVI.Phase = kargoapi.VerificationPhaseError
		newVI.Message = fmt.Errorf(
			"error building AnalysisRun for Stage %q and Freight collection %q in namespace %q: %w",
			stage.Name,
			freight.ID,
			stage.Namespace,
			err,
		).Error()
		return newVI, nil
	}
	if err = r.client.Create(ctx, ar); err != nil {
		newVI.FinishTime = ptr.To(metav1.Now())
		newVI.Phase = kargoapi.VerificationPhaseError
		newVI.Message = fmt.Errorf(
			"error creating AnalysisRun %q in namespace %q: %w",
			ar.Name,
			ar.Namespace,
			err,
		).Error()
		return newVI, kubeclient.IgnoreInvalid(err) // Ignore errors which are due to validation issues
	}

	// Mark the verification as pending.
	newVI.FinishTime = ptr.To(ar.CreationTimestamp)
	newVI.Phase = kargoapi.VerificationPhasePending
	newVI.AnalysisRun = &kargoapi.AnalysisRunReference{
		Name:      ar.Name,
		Namespace: ar.Namespace,
		Phase:     string(ar.Status.Phase),
	}
	return newVI, nil
}

// getVerificationResult gets the result of the verification for the current
// Freight of a Stage.
//
// If the Stage does not have an AnalysisRun associated with the verification,
// an error is returned.
//
// If the Rollouts integration is disabled, then the verification is marked as
// failed with an appropriate message.
func (r *RegularStageReconciler) getVerificationResult(
	ctx context.Context,
	freight kargoapi.FreightCollection,
) (*kargoapi.VerificationInfo, error) {
	// Ensure all necessary information is available to get the verification.
	currentVI := freight.VerificationHistory.Current()
	if currentVI == nil {
		return nil, fmt.Errorf("no current verification info for Freight collection %q", freight.ID)
	}
	if currentVI.AnalysisRun == nil {
		return nil, fmt.Errorf(
			"no AnalysisRun reference in current verification info for Freight collection %q",
			freight.ID,
		)
	}

	// If the Rollouts integration is disabled, then we cannot get the
	// verification.
	if !r.cfg.RolloutsIntegrationEnabled {
		return &kargoapi.VerificationInfo{
			ID:         currentVI.ID,
			StartTime:  currentVI.StartTime,
			FinishTime: ptr.To(metav1.Now()),
			Phase:      kargoapi.VerificationPhaseError,
			Message:    "Rollouts integration is disabled on this controller: cannot get verification result",
		}, nil
	}

	// TODO(hidde): This retry logic has been put in place because we have
	// observed the cache not being up-to-date with the API server in some
	// edge case scenarios. While this is not a long-term solution, it cures
	// the symptoms for now. We should investigate the root cause of this
	// issue and remove this retry logic when the root cause has been resolved.
	ar := rolloutsapi.AnalysisRun{}
	if err := retry.OnError(r.backoffCfg, func(err error) bool {
		return apierrors.IsNotFound(err)
	}, func() error {
		return r.client.Get(ctx, types.NamespacedName{
			Namespace: currentVI.AnalysisRun.Namespace,
			Name:      currentVI.AnalysisRun.Name,
		}, &ar)
	}); err != nil {
		return &kargoapi.VerificationInfo{
			ID:         currentVI.ID,
			Actor:      currentVI.Actor,
			StartTime:  currentVI.StartTime,
			FinishTime: currentVI.FinishTime,
			Phase:      kargoapi.VerificationPhaseError,
			Message: fmt.Errorf(
				"error getting AnalysisRun %q in namespace %q: %w",
				currentVI.AnalysisRun.Name,
				currentVI.AnalysisRun.Namespace,
				err,
			).Error(),
			AnalysisRun: currentVI.AnalysisRun.DeepCopy(),
		}, err
	}

	// Return a new VerificationInfo with the same ID and the information from
	// the current state of the AnalysisRun.
	return &kargoapi.VerificationInfo{
		ID:         currentVI.ID,
		Actor:      currentVI.Actor,
		StartTime:  currentVI.StartTime,
		FinishTime: ar.Status.CompletedAt(),
		Phase:      kargoapi.VerificationPhase(ar.Status.Phase),
		Message:    ar.Status.Message,
		AnalysisRun: &kargoapi.AnalysisRunReference{
			Name:      ar.Name,
			Namespace: ar.Namespace,
			Phase:     string(ar.Status.Phase),
		},
	}, nil
}

// abortVerification aborts the verification for the current Freight of a Stage.
func (r *RegularStageReconciler) abortVerification(
	ctx context.Context,
	freight kargoapi.FreightCollection,
	req *kargoapi.VerificationRequest,
) (*kargoapi.VerificationInfo, error) {
	// Ensure all necessary information is available to abort the verification.
	currentVI := freight.VerificationHistory.Current()
	if currentVI == nil {
		return nil, fmt.Errorf("no current verification info for Freight collection %q", freight.ID)
	}
	if currentVI.AnalysisRun == nil {
		return nil, fmt.Errorf(
			"no AnalysisRun reference in current verification info for Freight collection %q",
			freight.ID,
		)
	}

	// If the current verification is already terminal, then there is no need
	// to abort it.
	if currentVI.Phase.IsTerminal() {
		return currentVI, nil
	}

	// Determine the actor who requested the abort.
	actor := currentVI.Actor
	if req.ForID(currentVI.ID) {
		actor = req.Actor
	}

	// If the Rollouts integration is disabled, then we cannot abort the
	// verification.
	if !r.cfg.RolloutsIntegrationEnabled {
		return &kargoapi.VerificationInfo{
			ID:          currentVI.ID,
			Actor:       actor,
			StartTime:   currentVI.StartTime,
			FinishTime:  ptr.To(metav1.Now()),
			Phase:       kargoapi.VerificationPhaseError,
			Message:     "Rollouts integration is disabled on this controller: cannot abort verification",
			AnalysisRun: currentVI.AnalysisRun.DeepCopy(),
		}, nil
	}

	// Patch the AnalysisRun to request the abort.
	ar := &rolloutsapi.AnalysisRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: currentVI.AnalysisRun.Namespace,
			Name:      currentVI.AnalysisRun.Name,
		},
	}
	if err := r.client.Patch(
		ctx,
		ar,
		client.RawPatch(types.MergePatchType, []byte(`{"spec":{"terminate":true}}`)),
	); err != nil {
		// TODO(hidde): we should consider better error handling here to e.g.
		// retry an abort request when the Kubernetes API server is under heavy
		// load.
		return &kargoapi.VerificationInfo{
			ID:         currentVI.ID,
			Actor:      actor,
			StartTime:  currentVI.StartTime,
			FinishTime: ptr.To(metav1.Now()),
			Phase:      kargoapi.VerificationPhaseError,
			Message: fmt.Errorf(
				"error terminating AnalysisRun %q in namespace %q: %w", ar.Name, ar.Namespace, err,
			).Error(),
			AnalysisRun: currentVI.AnalysisRun.DeepCopy(),
		}, nil
	}

	// Return a new VerificationInfo with the same ID and a message indicating
	// that the verification was aborted. The Phase will be set to Failed, as
	// the verification was not successful.
	// We do not use the further information from the AnalysisRun, as this
	// will indicate a "Succeeded" phase due to Argo Rollouts behavior.
	return &kargoapi.VerificationInfo{
		ID:          currentVI.ID,
		Actor:       actor,
		StartTime:   currentVI.StartTime,
		FinishTime:  ptr.To(metav1.Now()),
		Phase:       kargoapi.VerificationPhaseFailed,
		Message:     "Verification aborted by user",
		AnalysisRun: currentVI.AnalysisRun.DeepCopy(),
	}, nil
}

// findExistingAnalysisRun finds the most recent AnalysisRun for a Stage and
// Freight collection in the namespace of the Stage. If no AnalysisRun is found,
// it returns nil.
func (r *RegularStageReconciler) findExistingAnalysisRun(
	ctx context.Context,
	stage types.NamespacedName,
	freightColID string,
) (*rolloutsapi.AnalysisRun, error) {
	analysisRuns := &rolloutsapi.AnalysisRunList{}
	if err := r.client.List(
		ctx,
		analysisRuns,
		client.InNamespace(stage.Namespace),
		client.MatchingLabelsSelector{
			Selector: labels.SelectorFromSet(map[string]string{
				kargoapi.StageLabelKey:             stage.Name,
				kargoapi.FreightCollectionLabelKey: freightColID,
			}),
		},
	); err != nil {
		return nil, fmt.Errorf(
			"error listing AnalysisRuns for Stage %q and Freight collection %q in namespace %q: %w",
			stage.Name, freightColID, stage.Namespace, err,
		)
	}

	if len(analysisRuns.Items) == 0 {
		return nil, nil
	}

	// Sort the AnalysisRuns by creation timestamp, so that the most recent
	// one is first.
	slices.SortFunc(analysisRuns.Items, func(lhs, rhs rolloutsapi.AnalysisRun) int {
		return rhs.CreationTimestamp.Time.Compare(lhs.CreationTimestamp.Time)
	})
	return &analysisRuns.Items[0], nil
}

// autoPromoteFreight automatically promotes the latest promotable (i.e.
// verified) Freight for a Stage if auto-promotion is allowed (see
// autoPromotionAllowed).
func (r *RegularStageReconciler) autoPromoteFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
) (kargoapi.StageStatus, error) {
	logger := logging.LoggerFromContext(ctx)
	newStatus := *stage.Status.DeepCopy()

	// If the Stage has no requested Freight, then there is nothing to promote.
	// NB: This should not happen in practice, as a Stage cannot exist without
	// requested Freight.
	if len(stage.Spec.RequestedFreight) == 0 {
		return newStatus, nil
	}

	stageRef := types.NamespacedName{Namespace: stage.Namespace, Name: stage.Name}

	// Confirm that auto-promotion is allowed for the Stage.
	if autoPromotionAllowed, err := r.autoPromotionAllowed(ctx, stageRef); err != nil || !autoPromotionAllowed {
		return newStatus, err
	}

	// Retrieve promotable Freight for the Stage.
	promotableFreight, err := r.getPromotableFreight(ctx, stage)
	if err != nil {
		return newStatus, err
	}

	// If the Stage has no current Freight, then we can promote any available
	currentFreight := newStatus.FreightHistory.Current()

	// Check if there is any new Freight which can be auto-promoted.
	for origin, freight := range promotableFreight {
		if len(freight) == 0 {
			logger.Debug("no Freight from origin available for auto-promotion", "origin", origin)
			continue
		}

		// Find the latest Freight by sorting the available Freight by creation time
		// in descending order.
		slices.SortFunc(freight, func(lhs, rhs kargoapi.Freight) int {
			return rhs.CreationTimestamp.Time.Compare(lhs.CreationTimestamp.Time)
		})
		latestFreight := freight[0]

		freightLogger := logger.WithValues("origin", origin, "freight", latestFreight.Name)

		// Only proceed if the latest available Freight is different from the
		// current Freight in the Stage.
		if currentFreight != nil && len(currentFreight.Freight) > 0 {
			if freightRef, ok := currentFreight.Freight[origin]; ok && freightRef.Name == latestFreight.Name {
				freightLogger.Debug("Stage already has latest available Freight for origin")
				continue
			}
		}

		// If a Promotion already exists for this Stage and Freight, then we
		// should not create a new one.
		promotions := &kargoapi.PromotionList{}
		if err = r.client.List(
			ctx,
			promotions,
			client.InNamespace(stage.Namespace),
			client.MatchingFieldsSelector{
				Selector: fields.OneTermEqualSelector(
					indexer.PromotionsByStageAndFreightField,
					indexer.StageAndFreightKey(stage.Name, latestFreight.Name),
				),
			},
			client.Limit(1),
		); err != nil {
			return newStatus, fmt.Errorf(
				"error listing existing Promotions for Freight %q in namespace %q: %w",
				latestFreight.Name, stage.Namespace, err,
			)
		}
		if len(promotions.Items) > 0 {
			freightLogger.Debug("Promotion already exists for Freight")
			continue
		}

		// Auto promote the latest available Freight and record an event.
		promotion := kargo.NewPromotion(ctx, *stage, latestFreight.Name)
		if err := r.client.Create(ctx, &promotion); err != nil {
			return newStatus, fmt.Errorf(
				"error creating Promotion for Freight %q in namespace %q: %w",
				latestFreight.Name, stage.Namespace, err,
			)
		}
		r.eventRecorder.AnnotatedEventf(
			&promotion,
			kargoEvent.NewPromotionAnnotations(
				ctx,
				kargoapi.FormatEventControllerActor(r.cfg.Name()),
				&promotion,
				&latestFreight,
			),
			corev1.EventTypeNormal,
			kargoapi.EventReasonPromotionCreated,
			"Automatically promoted Freight from origin %q for Stage %q",
			origin,
			promotion.Spec.Stage,
		)
		logger.Debug(
			"created Promotion resource",
			"promotion", promotion.Name,
		)
	}

	return newStatus, nil
}

// autoPromotionAllowed checks if auto-promotion is allowed for the given Stage.
func (r *RegularStageReconciler) autoPromotionAllowed(
	ctx context.Context,
	stage types.NamespacedName,
) (bool, error) {
	logger := logging.LoggerFromContext(ctx)

	project := &kargoapi.Project{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: stage.Namespace}, project); err != nil {
		return false, fmt.Errorf("error getting Project %q in namespace %q: %w", stage.Name, stage.Namespace, err)
	}

	if project.Spec == nil || len(project.Spec.PromotionPolicies) == 0 {
		logger.Debug("found no PromotionPolicy associated with Stage")
		return false, nil
	}

	for _, policy := range project.Spec.PromotionPolicies {
		if policy.Stage == stage.Name {
			logger.Debug(
				"found PromotionPolicy associated with Stage",
				"autoPromotionEnabled", policy.AutoPromotionEnabled,
			)
			return policy.AutoPromotionEnabled, nil
		}
	}

	logger.Debug("found no PromotionPolicy associated with Stage")
	return false, nil
}

// getPromotableFreight retrieves a map of []Freight promotable to the specified
// Stage, indexed by origin.
func (r *RegularStageReconciler) getPromotableFreight(
	ctx context.Context,
	stage *kargoapi.Stage,
) (map[string][]kargoapi.Freight, error) {
	availableFreight, err := stage.ListAvailableFreight(ctx, r.client)
	if err != nil {
		return nil, fmt.Errorf(
			"error listing available Freight for Stage %q: %w",
			stage.Name, err,
		)
	}

	var promotableFreight = make(map[string][]kargoapi.Freight)
	for _, freight := range availableFreight {
		originID := freight.Origin.String()
		if _, ok := promotableFreight[originID]; !ok {
			promotableFreight[originID] = []kargoapi.Freight{freight}
		} else {
			promotableFreight[originID] = append(promotableFreight[originID], freight)
		}
	}

	return promotableFreight, nil
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
	if err := kargoapi.RemoveFinalizer(ctx, r.client, stage); err != nil {
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

// summarizeConditions summarizes the conditions of the given Stage. It sets the
// Ready condition based on the Promoting, Healthy and Verified conditions.
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

		// Backwards compatibility: set the Phase and Message.
		// TODO: Remove this in a future release.
		newStatus.Phase = kargoapi.StagePhaseFailed
		newStatus.Message = err.Error()

		return
	}

	// By default, the Stage is steady unless we find a more specific condition.
	// TODO: Remove this in a future release.
	newStatus.Phase = kargoapi.StagePhaseSteady

	// Backwards compatibility: clear the Message field of the Status
	// and set the Freight summary.
	// TODO: Remove this in a future release.
	newStatus.Message = ""
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

		// Backwards compatibility: set Phase to Promoting.
		// TODO: Remove this in a future release.
		newStatus.Phase = kargoapi.StagePhasePromoting

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

		// Backwards compatibility: set Phase to Failed.
		// TODO: Remove this in a future release.
		newStatus.Phase = kargoapi.StagePhaseFailed

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

			if healthCond.Status == metav1.ConditionFalse {
				// Backwards compatibility: set Phase to Failed on health failure.
				// TODO: Remove this in a future release.
				newStatus.Phase = kargoapi.StagePhaseFailed
			}
		}
		conditions.Set(newStatus, readyCond)

		return
	}

	// If we are not verified, then we are not Ready.
	verificationCond := conditions.Get(newStatus, kargoapi.ConditionTypeVerified)
	if verificationCond == nil || verificationCond.Status != metav1.ConditionTrue {
		// Backwards compatibility: set Phase to Verifying.
		// TODO: Remove this in a future release.
		newStatus.Phase = kargoapi.StagePhaseVerifying

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

			// Backwards compatibility: set Phase to Failed on verification failure.
			// TODO: Remove this in a future release.
			if verificationCond.Status == metav1.ConditionFalse {
				newStatus.Phase = kargoapi.StagePhaseFailed
			}
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

	// Backwards compatibility: set Phase to Steady.
	// TODO: Remove this in a future release.
	newStatus.Phase = kargoapi.StagePhaseSteady
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
