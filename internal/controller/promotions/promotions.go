package promotions

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/promotion"
	"github.com/akuity/kargo/internal/controller/runtime"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/kargo"
	"github.com/akuity/kargo/internal/kubeclient"
	libEvent "github.com/akuity/kargo/internal/kubernetes/event"
	"github.com/akuity/kargo/internal/logging"
)

// ReconcilerConfig represents configuration for the promotion reconciler.
type ReconcilerConfig struct {
	ShardName string `envconfig:"SHARD_NAME"`
}

func (c ReconcilerConfig) Name() string {
	name := "promotion-controller"
	if c.ShardName != "" {
		return name + "-" + c.ShardName
	}
	return name
}

func ReconcilerConfigFromEnv() ReconcilerConfig {
	var cfg ReconcilerConfig
	envconfig.MustProcess("", &cfg)
	return cfg
}

// reconciler reconciles Promotion resources.
type reconciler struct {
	kargoClient     client.Client
	promoMechanisms promotion.Mechanism

	cfg ReconcilerConfig

	recorder record.EventRecorder

	pqs            *promoQueues
	initializeOnce sync.Once

	// The following behaviors are overridable for testing purposes:

	getStageFn func(
		context.Context,
		client.Client,
		types.NamespacedName,
	) (*kargoapi.Stage, error)

	promoteFn func(context.Context, kargoapi.Promotion, *kargoapi.Freight) (*kargoapi.PromotionStatus, error)
}

// SetupReconcilerWithManager initializes a reconciler for Promotion resources
// and registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	argocdMgr manager.Manager,
	credentialsDB credentials.Database,
	cfg ReconcilerConfig,
) error {
	// Index running Promotions by Argo CD Applications
	if err := kubeclient.IndexRunningPromotionsByArgoCDApplications(ctx, kargoMgr, cfg.ShardName); err != nil {
		return fmt.Errorf("index running Promotions by Argo CD Applications: %w", err)
	}

	shardPredicate, err := controller.GetShardPredicate(cfg.ShardName)
	if err != nil {
		return fmt.Errorf("error creating shard selector predicate: %w", err)
	}
	shardRequirement, err := controller.GetShardRequirement(cfg.ShardName)
	if err != nil {
		return fmt.Errorf("error creating shard requirement: %w", err)
	}
	shardSelector := labels.NewSelector().Add(*shardRequirement)

	var argocdClient client.Client
	if argocdMgr != nil {
		argocdClient = argocdMgr.GetClient()
	}

	reconciler := newReconciler(
		kargoMgr.GetClient(),
		argocdClient,
		libEvent.NewRecorder(ctx, kargoMgr.GetScheme(), kargoMgr.GetClient(), cfg.Name()),
		credentialsDB,
		cfg,
	)

	c, err := ctrl.NewControllerManagedBy(kargoMgr).
		For(&kargoapi.Promotion{}).
		WithEventFilter(predicate.Or(
			predicate.GenerationChangedPredicate{},
			kargo.RefreshRequested{},
		)).
		WithEventFilter(shardPredicate).
		WithOptions(controller.CommonOptions()).
		Build(reconciler)
	if err != nil {
		return fmt.Errorf("error building Promotion controller: %w", err)
	}

	logger := logging.LoggerFromContext(ctx)

	// If Argo CD integration is disabled, this manager will be nil and we won't
	// care about this watch anyway.
	if argocdMgr != nil {
		if err := c.Watch(
			source.Kind(
				argocdMgr.GetCache(),
				&argocd.Application{},
				&UpdatedArgoCDAppHandler[*argocd.Application]{
					kargoClient:   kargoMgr.GetClient(),
					shardSelector: shardSelector,
				},
				ArgoCDAppOperationCompleted[*argocd.Application]{
					logger: logger,
				},
			),
		); err != nil {
			return fmt.Errorf("unable to watch Applications: %w", err)
		}
	}

	// Watch Promotions that complete and enqueue the next highest promotion key
	priorityQueueHandler := &EnqueueHighestPriorityPromotionHandler[*kargoapi.Promotion]{
		ctx:         ctx,
		logger:      logger,
		kargoClient: reconciler.kargoClient,
		pqs:         reconciler.pqs,
	}
	promoWentTerminal := kargo.NewPromoWentTerminalPredicate(logger)
	if err := c.Watch(
		source.Kind(
			kargoMgr.GetCache(),
			&kargoapi.Promotion{},
			priorityQueueHandler,
			promoWentTerminal,
		),
	); err != nil {
		return fmt.Errorf("unable to watch Promotions: %w", err)
	}

	return nil
}

func newReconciler(
	kargoClient client.Client,
	argocdClient client.Client,
	recorder record.EventRecorder,
	credentialsDB credentials.Database,
	cfg ReconcilerConfig,
) *reconciler {
	pqs := promoQueues{
		activePromoByStage:        map[types.NamespacedName]string{},
		pendingPromoQueuesByStage: map[types.NamespacedName]runtime.PriorityQueue{},
	}
	r := &reconciler{
		kargoClient: kargoClient,
		recorder:    recorder,
		cfg:         cfg,
		pqs:         &pqs,
		promoMechanisms: promotion.NewMechanisms(
			argocdClient,
			credentialsDB,
		),
	}
	r.getStageFn = kargoapi.GetStage
	r.promoteFn = r.promote
	return r
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := logging.LoggerFromContext(ctx).
		WithFields(log.Fields{
			"namespace": req.NamespacedName.Namespace,
			"promotion": req.NamespacedName.Name,
		})
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling Promotion")

	// Note that initialization occurs here because we basically know that the
	// controller runtime client's cache is ready at this point. We cannot attempt
	// to list Promotions prior to that point.
	var err error
	r.initializeOnce.Do(func() {
		promos := kargoapi.PromotionList{}
		if err = r.kargoClient.List(ctx, &promos); err != nil {
			err = fmt.Errorf("error listing promotions: %w", err)
		} else {
			r.pqs.initializeQueues(ctx, promos)
			logger.Debug(
				"initialized Stage-specific Promotion queues from list of existing Promotions",
			)
		}
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error initializing Promotion queues: %w", err)
	}

	// Find the Promotion
	promo, err := kargoapi.GetPromotion(ctx, r.kargoClient, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, err
	}
	if promo == nil || promo.Status.Phase.IsTerminal() {
		// Ignore if not found or already finished. Promo might be nil if the
		// Promotion was deleted after the current reconciliation request was issued.
		return ctrl.Result{}, nil
	}
	// Find the Freight
	freight, err := kargoapi.GetFreight(ctx, r.kargoClient, types.NamespacedName{
		Namespace: promo.Namespace,
		Name:      promo.Spec.Freight,
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf(
			"error finding Freight %q in namespace %q: %w",
			promo.Spec.Freight,
			promo.Namespace,
			err,
		)
	}

	logger = logger.WithFields(log.Fields{
		"namespace": req.NamespacedName.Namespace,
		"promotion": req.NamespacedName.Name,
		"stage":     promo.Spec.Stage,
		"freight":   promo.Spec.Freight,
	})

	if promo.Status.Phase == kargoapi.PromotionPhaseRunning {
		// anything we've already marked Running, we allow it to continue to reconcile
		logger.Debug("continuing Promotion")
	} else {
		// promo is Pending. Try to begin it.
		if !r.pqs.tryBegin(ctx, promo) {
			// It wasn't our turn. Mark this promo as Pending (if it wasn't already)
			if promo.Status.Phase != kargoapi.PromotionPhasePending {
				err = kubeclient.PatchStatus(ctx, r.kargoClient, promo, func(status *kargoapi.PromotionStatus) {
					status.Phase = kargoapi.PromotionPhasePending
				})
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		logger.Info("began promotion")
	}

	// Update promo status as Running to give visibility in UI. Also, a promo which
	// has already entered Running status will be allowed to continue to reconcile.
	if promo.Status.Phase != kargoapi.PromotionPhaseRunning {
		if err = kubeclient.PatchStatus(ctx, r.kargoClient, promo, func(status *kargoapi.PromotionStatus) {
			status.Phase = kargoapi.PromotionPhaseRunning
		}); err != nil {
			return ctrl.Result{}, err
		}
	}

	promoCtx := logging.ContextWithLogger(ctx, logger)

	newStatus := promo.Status.DeepCopy()

	// Wrap the promoteFn() call in an anonymous function to recover() any panics, so
	// we can update the promo's phase with Error if it does. This breaks an infinite
	// cycle of a bad promo continuously failing to reconcile, and surfaces the error.
	func() {
		defer func() {
			if err := recover(); err != nil {
				logger.Errorf("Promotion panic: %v", err)
				newStatus.Phase = kargoapi.PromotionPhaseErrored
				newStatus.Message = fmt.Sprintf("%v", err)
			}
		}()
		otherStatus, promoteErr := r.promoteFn(
			promoCtx,
			*promo,
			freight,
		)
		if promoteErr != nil {
			newStatus.Phase = kargoapi.PromotionPhaseErrored
			newStatus.Message = promoteErr.Error()
			logger.Errorf("error executing Promotion: %s", promoteErr)
		} else {
			newStatus = otherStatus
		}
	}()

	if newStatus.Phase.IsTerminal() {
		logger.Infof("promotion %s", newStatus.Phase)
	}

	// Record the current refresh token as having been handled.
	if token, ok := kargoapi.RefreshAnnotationValue(promo.GetAnnotations()); ok {
		newStatus.LastHandledRefresh = token
	}

	err = kubeclient.PatchStatus(ctx, r.kargoClient, promo, func(status *kargoapi.PromotionStatus) {
		*status = *newStatus
	})
	if err != nil {
		logger.Errorf("error updating Promotion status: %s", err)
	}

	// Record event after patching status if new phase is terminal
	if newStatus.Phase.IsTerminal() {
		stage, getStageErr := r.getStageFn(
			ctx,
			r.kargoClient,
			types.NamespacedName{
				Namespace: promo.Namespace,
				Name:      promo.Spec.Stage,
			},
		)
		if getStageErr != nil {
			return ctrl.Result{}, fmt.Errorf("get stage: %w", err)
		}
		if stage == nil {
			return ctrl.Result{}, fmt.Errorf(
				"stage %q not found in namespace %q",
				promo.Spec.Stage,
				promo.Namespace,
			)
		}

		var reason string
		switch newStatus.Phase {
		case kargoapi.PromotionPhaseSucceeded:
			reason = kargoapi.EventReasonPromotionSucceeded
		case kargoapi.PromotionPhaseFailed:
			reason = kargoapi.EventReasonPromotionFailed
		case kargoapi.PromotionPhaseErrored:
			reason = kargoapi.EventReasonPromotionErrored
		}

		msg := fmt.Sprintf("Promotion %s", newStatus.Phase)
		if newStatus.Message != "" {
			msg += fmt.Sprintf(": %s", newStatus.Message)
		}

		eventAnnotations := kargoapi.NewPromotionEventAnnotations(ctx,
			kargoapi.FormatEventControllerActor(r.cfg.Name()),
			promo, freight)

		if newStatus.Phase == kargoapi.PromotionPhaseSucceeded {
			eventAnnotations[kargoapi.AnnotationKeyEventVerificationPending] =
				strconv.FormatBool(stage.Spec.Verification != nil)
		}
		r.recorder.AnnotatedEventf(promo, eventAnnotations, corev1.EventTypeNormal, reason, msg)
	}

	if err != nil {
		// Controller runtime automatically gives us a progressive backoff if err is
		// not nil
		return ctrl.Result{}, err
	}

	// If the promotion is still running, we'll need to periodically check on
	// it.
	//
	// TODO: Make this configurable
	if newStatus.Phase == kargoapi.PromotionPhaseRunning {
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}
	return ctrl.Result{}, nil
}

func (r *reconciler) promote(
	ctx context.Context,
	promo kargoapi.Promotion,
	targetFreight *kargoapi.Freight,
) (*kargoapi.PromotionStatus, error) {
	logger := logging.LoggerFromContext(ctx)
	stageName := promo.Spec.Stage
	stageNamespace := promo.Namespace

	stage, err := r.getStageFn(
		ctx,
		r.kargoClient,
		types.NamespacedName{
			Namespace: stageNamespace,
			Name:      stageName,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error finding Stage %q in namespace %q: %w", stageName, stageNamespace, err)
	}
	if stage == nil {
		return nil, fmt.Errorf("could not find Stage %q in namespace %q", stageName, stageNamespace)
	}
	logger.Debug("found associated Stage")

	if targetFreight == nil {
		return nil, fmt.Errorf("Freight %q not found in namespace %q", promo.Spec.Freight, promo.Namespace)
	}
	upstreamStages := make([]string, len(stage.Spec.Subscriptions.UpstreamStages))
	for i, upstreamStage := range stage.Spec.Subscriptions.UpstreamStages {
		upstreamStages[i] = upstreamStage.Name
	}
	if !kargoapi.IsFreightAvailable(targetFreight, stageName, upstreamStages) {
		return nil, fmt.Errorf(
			"Freight %q is not available to Stage %q in namespace %q",
			promo.Spec.Freight,
			stageName,
			stageNamespace,
		)
	}

	logger = logger.WithField("targetFreight", targetFreight.Name)

	targetFreightRef := kargoapi.FreightReference{
		Name:      targetFreight.Name,
		Commits:   targetFreight.Commits,
		Images:    targetFreight.Images,
		Charts:    targetFreight.Charts,
		Warehouse: targetFreight.Warehouse,
	}
	err = kubeclient.PatchStatus(ctx, r.kargoClient, stage, func(status *kargoapi.StageStatus) {
		status.Phase = kargoapi.StagePhasePromoting
		status.CurrentPromotion = &kargoapi.PromotionInfo{
			Name:    promo.Name,
			Freight: targetFreightRef,
		}
	})
	if err != nil {
		return nil, err
	}

	newStatus, nextFreight, err := r.promoMechanisms.Promote(ctx, stage, &promo, targetFreightRef)
	if err != nil {
		return nil, err
	}
	newStatus.Freight = &nextFreight

	logger.Debugf("promotion %s", newStatus.Phase)

	if newStatus.Phase.IsTerminal() {
		// The assumption is that controller does not process multiple promotions in one stage
		// so we are safe from race conditions and can just update the status
		// TODO: remove all patching of Stage status out of promo reconciler
		if err = kubeclient.PatchStatus(ctx, r.kargoClient, stage, func(status *kargoapi.StageStatus) {
			status.LastPromotion = status.CurrentPromotion
			status.LastPromotion.Status = newStatus
			if newStatus.Phase == kargoapi.PromotionPhaseSucceeded {
				// Handle specific things that need to happen on success.
				// 1. Trigger re-verification for re-promotions.
				// 2. Otherwise, update the current freight and history.
				// 3. Update the phase to Verifying and clear the current promotion.
				if status.CurrentFreight != nil &&
					status.CurrentFreight.Name == targetFreight.Name {
					if err = kargoapi.ReverifyStageFreight(
						ctx,
						r.kargoClient,
						types.NamespacedName{
							Namespace: stageNamespace,
							Name:      stageName,
						},
					); err != nil {
						// Log the error, but don't let failure to initiate re-verification
						// prevent the promotion from succeeding.
						logger.Errorf("error triggering re-verification: %s", err)
					}
				} else if stage.Spec.PromotionMechanisms != nil {
					status.CurrentFreight = &nextFreight
					status.History.UpdateOrPush(nextFreight)
				}
				status.Phase = kargoapi.StagePhaseVerifying
				status.CurrentPromotion = nil
			}
		}); err != nil {
			return nil, fmt.Errorf(
				"error updating status of Stage %q in namespace %q: %w",
				stageName,
				stageNamespace,
				err,
			)
		}
	}

	return newStatus, nil
}
