package promotions

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
	"github.com/akuity/kargo/internal/controller/promotion"
	"github.com/akuity/kargo/internal/controller/runtime"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/kargo"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
)

// reconciler reconciles Promotion resources.
type reconciler struct {
	kargoClient     client.Client
	promoMechanisms promotion.Mechanism

	pqs            *promoQueues
	initializeOnce sync.Once

	// The following behaviors are overridable for testing purposes:

	promoteFn func(context.Context, kargoapi.Promotion) error
}

// SetupReconcilerWithManager initializes a reconciler for Promotion resources
// and registers it with the provided Manager.
func SetupReconcilerWithManager(
	ctx context.Context,
	kargoMgr manager.Manager,
	argoMgr manager.Manager,
	credentialsDB credentials.Database,
	shardName string,
) error {

	shardPredicate, err := controller.GetShardPredicate(shardName)
	if err != nil {
		return errors.Wrap(err, "error creating shard selector predicate")
	}

	reconciler := newReconciler(
		kargoMgr.GetClient(),
		argoMgr.GetClient(),
		credentialsDB,
	)

	changePredicate := predicate.Or(
		predicate.GenerationChangedPredicate{},
		predicate.AnnotationChangedPredicate{},
	)

	c, err := ctrl.NewControllerManagedBy(kargoMgr).
		For(&kargoapi.Promotion{}).
		WithEventFilter(changePredicate).
		WithEventFilter(shardPredicate).
		WithEventFilter(kargo.IgnoreClearRefreshUpdates{}).
		WithOptions(controller.CommonOptions()).
		Build(reconciler)
	if err != nil {
		return errors.Wrap(err, "error building Promotion reconciler")
	}

	logger := logging.LoggerFromContext(ctx)
	// Watch Promotions that complete and enqueue the next highest promotion key
	priorityQueueHandler := &EnqueueHighestPriorityPromotionHandler{
		ctx:         ctx,
		logger:      logger,
		kargoClient: reconciler.kargoClient,
		pqs:         reconciler.pqs,
	}
	promoWentTerminal := kargo.NewPromoWentTerminalPredicate(logger)
	if err := c.Watch(&source.Kind{Type: &kargoapi.Promotion{}}, priorityQueueHandler, promoWentTerminal); err != nil {
		return errors.Wrap(err, "unable to watch Promotions")
	}

	return nil
}

func newReconciler(
	kargoClient client.Client,
	argoClient client.Client,
	credentialsDB credentials.Database,
) *reconciler {
	pqs := promoQueues{
		activePromoByStage:        map[types.NamespacedName]string{},
		pendingPromoQueuesByStage: map[types.NamespacedName]runtime.PriorityQueue{},
	}
	r := &reconciler{
		kargoClient: kargoClient,
		pqs:         &pqs,
		promoMechanisms: promotion.NewMechanisms(
			argoClient,
			credentialsDB,
		),
	}
	r.promoteFn = r.promote
	return r
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	result := ctrl.Result{
		// Note: If there is a failure, controller runtime ignores this and uses
		// progressive backoff instead. So this value only prevents requeueing
		// a Promotion if THIS reconciliation succeeds.
		RequeueAfter: 0,
	}

	logger := logging.LoggerFromContext(ctx)

	// Note that initialization occurs here because we basically know that the
	// controller runtime client's cache is ready at this point. We cannot attempt
	// to list Promotions prior to that point.
	var err error
	r.initializeOnce.Do(func() {
		promos := kargoapi.PromotionList{}
		if err = r.kargoClient.List(ctx, &promos); err != nil {
			err = errors.Wrap(err, "error listing promotions")
		} else {
			r.pqs.initializeQueues(ctx, promos)
			logger.Debug(
				"initialized Stage-specific Promotion queues from list of existing Promotions",
			)
		}
	})
	if err != nil {
		return result, errors.Wrap(err, "error initializing Promotion queues")
	}

	logger = logger.WithFields(log.Fields{
		"namespace": req.NamespacedName.Namespace,
		"promotion": req.NamespacedName.Name,
	})
	ctx = logging.ContextWithLogger(ctx, logger)
	logger.Debug("reconciling Promotion")

	// Find the Promotion
	promo, err := kargoapi.GetPromotion(ctx, r.kargoClient, req.NamespacedName)
	if err != nil {
		return result, err
	}
	if promo == nil {
		// Ignore if not found. This can happen if the Promotion was deleted after
		// the current reconciliation request was issued.
		return result, nil
	}

	if promo.Status.Phase == kargoapi.PromotionPhaseRunning {
		// anything we've already marked Running, we allow it to continue to reconcile
	} else if promo.Status.Phase.IsTerminal() {
		// if promo is already finished, nothing to do
		return result, nil
	} else {
		// promo is Pending. Try to begin it.
		if !r.pqs.tryBegin(ctx, promo) {
			// It wasn't our turn. Mark this promo as Pending (if it wasn't already)
			if promo.Status.Phase != kargoapi.PromotionPhasePending {
				err = kubeclient.PatchStatus(ctx, r.kargoClient, promo, func(status *kargoapi.PromotionStatus) {
					status.Phase = kargoapi.PromotionPhasePending
				})
				return result, err
			}
			return result, nil
		}
	}

	logger = logger.WithFields(log.Fields{
		"stage":   promo.Spec.Stage,
		"freight": promo.Spec.Freight,
	})
	logger.Debug("executing Promotion")

	// Update promo status as Running to give visibility in UI. Also, a promo which
	// has already entered Running status will be allowed to continue to reconcile.
	if promo.Status.Phase != kargoapi.PromotionPhaseRunning {
		if err = kubeclient.PatchStatus(ctx, r.kargoClient, promo, func(status *kargoapi.PromotionStatus) {
			status.Phase = kargoapi.PromotionPhaseRunning
		}); err != nil {
			return result, err
		}
	}

	promoCtx := logging.ContextWithLogger(ctx, logger)

	phase := kargoapi.PromotionPhaseSucceeded
	phaseError := ""

	// Wrap the promoteFn() call in an anonymous function to recover() any panics, so
	// we can update the promo's phase with Error if it does. This breaks an infinite
	// cycle of a bad promo continuously failing to reconcile, and surfaces the error.
	func() {
		defer func() {
			if err := recover(); err != nil {
				logger.Errorf("Promotion panic: %v", err)
				phase = kargoapi.PromotionPhaseErrored
				phaseError = fmt.Sprintf("%v", err)
			}
		}()
		if err = r.promoteFn(
			promoCtx,
			*promo,
		); err != nil {
			phase = kargoapi.PromotionPhaseErrored
			phaseError = err.Error()
			logger.Errorf("error executing Promotion: %s", err)
		}
	}()

	if phase.IsTerminal() {
		logger.Debugf("promotion %s", phase)
	}

	err = kubeclient.PatchStatus(ctx, r.kargoClient, promo, func(status *kargoapi.PromotionStatus) {
		status.Phase = phase
		status.Error = phaseError
	})
	if err != nil {
		logger.Errorf("error updating Promotion status: %s", err)
	}

	// Controller runtime automatically gives us a progressive backoff if err is not nil
	return result, err
}

func (r *reconciler) promote(
	ctx context.Context,
	promo kargoapi.Promotion,
) error {
	logger := logging.LoggerFromContext(ctx)
	stageName := promo.Spec.Stage
	stageNamespace := promo.Namespace
	freightName := promo.Spec.Freight

	stage, err := kargoapi.GetStage(
		ctx,
		r.kargoClient,
		types.NamespacedName{
			Namespace: stageNamespace,
			Name:      stageName,
		},
	)
	if err != nil {
		return errors.Wrapf(
			err,
			"error finding Stage %q in namespace %q",
			stageName,
			stageNamespace,
		)
	}
	if stage == nil {
		return errors.Errorf(
			"could not find Stage %q in namespace %q",
			stageName,
			stageNamespace,
		)
	}
	logger.Debug("found associated Stage")

	if stage.Status.CurrentFreight != nil && stage.Status.CurrentFreight.ID == freightName {
		logger.Debug("Stage already has the desired Freight")
		return nil
	}

	targetFreight, err := kargoapi.GetFreight(
		ctx,
		r.kargoClient,
		types.NamespacedName{
			Namespace: promo.Namespace,
			Name:      promo.Spec.Freight,
		},
	)
	if err != nil {
		return errors.Wrapf(
			err,
			"error finding Freight %q in namespace %q",
			promo.Spec.Freight, promo.Namespace,
		)
	}
	if targetFreight == nil {
		return errors.Errorf(
			"Freight %q not found in namespace %q",
			promo.Spec.Freight,
			promo.Namespace,
		)
	}
	upstreamStages := make([]string, len(stage.Spec.Subscriptions.UpstreamStages))
	for i, upstreamStage := range stage.Spec.Subscriptions.UpstreamStages {
		upstreamStages[i] = upstreamStage.Name
	}
	if !kargoapi.IsFreightAvailable(targetFreight, stageName, upstreamStages) {
		return errors.Errorf(
			"Freight %q is not available to Stage %q in namespace %q",
			promo.Spec.Freight,
			stageName,
			stageNamespace,
		)
	}

	simpleTargetFreight := kargoapi.SimpleFreight{
		ID:      targetFreight.ID,
		Commits: targetFreight.Commits,
		Images:  targetFreight.Images,
		Charts:  targetFreight.Charts,
	}

	err = kubeclient.PatchStatus(ctx, r.kargoClient, stage, func(status *kargoapi.StageStatus) {
		status.CurrentPromotion = &kargoapi.PromotionInfo{
			Name:    promo.Name,
			Freight: simpleTargetFreight,
		}
	})
	if err != nil {
		return err
	}

	nextFreight, err := r.promoMechanisms.Promote(ctx, stage, simpleTargetFreight)
	if err != nil {
		return err
	}

	// The assumption is that controller does not process multiple promotions in one stage
	// so we are safe from race conditions and can just update the status
	err = kubeclient.PatchStatus(ctx, r.kargoClient, stage, func(status *kargoapi.StageStatus) {
		status.CurrentPromotion = nil
		// control-flow Stage history is maintained in Stage controller.
		// So we only modify history for normal Stages.
		// (Technically, we should prevent creating promotion jobs on
		// control-flow stages in the first place)
		if stage.Spec.PromotionMechanisms != nil {
			status.CurrentFreight = &nextFreight
			status.History.Push(nextFreight)
		}
	})

	return errors.Wrapf(
		err,
		"error updating status of Stage %q in namespace %q",
		stageName,
		stageNamespace,
	)
}
