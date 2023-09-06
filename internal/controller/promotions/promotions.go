package promotions

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/akuity/bookkeeper"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller"
	"github.com/akuity/kargo/internal/controller/promotion"
	"github.com/akuity/kargo/internal/controller/runtime"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/kubeclient"
	"github.com/akuity/kargo/internal/logging"
)

// reconciler reconciles Promotion resources.
type reconciler struct {
	kargoClient     client.Client
	promoMechanisms promotion.Mechanism

	promoQueuesByStage   map[types.NamespacedName]runtime.PriorityQueue
	promoQueuesByStageMu sync.Mutex
	initializeOnce       sync.Once

	// Overridable for testing:
	promoteFn func(
		ctx context.Context,
		stageName string,
		stageNamespace string,
		freightID string,
	) error
}

// SetupReconcilerWithManager initializes a reconciler for Promotion resources
// and registers it with the provided Manager.
func SetupReconcilerWithManager(
	kargoMgr manager.Manager,
	argoMgr manager.Manager,
	credentialsDB credentials.Database,
	bookkeeperService bookkeeper.Service,
	shardName string,
) error {

	shardPredicate, err := controller.GetShardPredicate(shardName)
	if err != nil {
		return errors.Wrap(err, "error creating shard selector predicate")
	}

	return errors.Wrap(
		ctrl.NewControllerManagedBy(kargoMgr).
			For(&kargoapi.Promotion{}).
			WithEventFilter(predicate.GenerationChangedPredicate{}).
			WithEventFilter(shardPredicate).
			Complete(
				newReconciler(
					kargoMgr.GetClient(),
					argoMgr.GetClient(),
					credentialsDB,
					bookkeeperService,
				),
			),
		"error registering Promotion reconciler",
	)
}

func newReconciler(
	kargoClient client.Client,
	argoClient client.Client,
	credentialsDB credentials.Database,
	bookkeeperService bookkeeper.Service,
) *reconciler {
	r := &reconciler{
		kargoClient:        kargoClient,
		promoQueuesByStage: map[types.NamespacedName]runtime.PriorityQueue{},
		promoMechanisms: promotion.NewMechanisms(
			argoClient,
			credentialsDB,
			bookkeeperService,
		),
	}
	r.promoteFn = r.promote
	return r
}

func newPromotionsQueue() runtime.PriorityQueue {
	// We can safely ignore errors here because the only error that can happen
	// involves initializing the queue with a nil priority function, which we
	// know we aren't doing.
	pq, _ := runtime.NewPriorityQueue(func(left, right client.Object) bool {
		return left.GetCreationTimestamp().Time.
			Before(right.GetCreationTimestamp().Time)
	})
	return pq
}

// Reconcile is part of the main Kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *reconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	// We count all of Reconcile() as a critical section of code to ensure we
	// don't start reconciling a second Promotion before lazy initialization
	// completes upon reconciliation of the FIRST promotion.
	r.promoQueuesByStageMu.Lock()
	defer r.promoQueuesByStageMu.Unlock()

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
		if err = r.initializeQueues(ctx); err == nil {
			logger.Debug(
				"initialized Stage-specific Promotion queues from list of " +
					"existing Promotions",
			)
		}
		// TODO: Do not hardcode this interval
		go r.serializedSync(ctx, 10*time.Second)
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
	promo, err := r.getPromo(ctx, req.NamespacedName)
	if err != nil {
		return result, err
	}
	if promo == nil {
		// Ignore if not found. This can happen if the Promotion was deleted after
		// the current reconciliation request was issued.
		return result, nil
	}

	newStatus := r.syncPromo(ctx, promo)

	updateErr := kubeclient.PatchStatus(ctx, r.kargoClient, promo, func(status *kargoapi.PromotionStatus) {
		*status = newStatus
	})
	if updateErr != nil {
		logger.Errorf("error updating Promotion status: %s", updateErr)
	}

	// If we had no error, but couldn't update, then we DO have an error. But we
	// do it this way so that a failure to update is never counted as THE failure
	// when something else more serious occurred first.
	if err == nil {
		err = updateErr
	}

	// Controller runtime automatically gives us a progressive backoff if err is
	// not nil
	return result, err
}

// initializeQueues lists all Promotions and adds them to relevant priority
// queues. This is intended to be invoked ONCE and the caller MUST ensure that.
// It is also assumed that the caller has already obtained a lock on
// promoQueuesByStageMu.
func (r *reconciler) initializeQueues(ctx context.Context) error {
	promos := kargoapi.PromotionList{}
	if err := r.kargoClient.List(ctx, &promos); err != nil {
		return errors.Wrap(err, "error listing promotions")
	}
	logger := logging.LoggerFromContext(ctx)
	for _, p := range promos.Items {
		promo := p // This is to sidestep implicit memory aliasing in this for loop
		if promo.Status.Phase.IsTerminal() {
			continue
		}
		if promo.Status.Phase == "" {
			if err := kubeclient.PatchStatus(ctx, r.kargoClient, &promo, func(status *kargoapi.PromotionStatus) {
				status.Phase = kargoapi.PromotionPhasePending
			}); err != nil {
				return errors.Wrapf(
					err,
					"error updating status of Promotion %q in namespace %q",
					promo.Name,
					promo.Namespace,
				)
			}
		}
		stage := types.NamespacedName{
			Namespace: promo.Namespace,
			Name:      promo.Spec.Stage,
		}
		pq, ok := r.promoQueuesByStage[stage]
		if !ok {
			pq = newPromotionsQueue()
			r.promoQueuesByStage[stage] = pq
		}
		// The only error that can occur here happens when you push a nil and we
		// know we're not doing that.
		pq.Push(&promo) // nolint: errcheck
		logger.WithFields(log.Fields{
			"promotion": promo.Name,
			"namespace": promo.Namespace,
			"stage":     promo.Spec.Stage,
			"phase":     promo.Status.Phase,
		}).Debug("pushed Promotion onto Stage-specific Promotion queue")
	}
	if logger.Logger.IsLevelEnabled(log.DebugLevel) {
		for stage, pq := range r.promoQueuesByStage {
			logger.WithFields(log.Fields{
				"stage":     stage.Name,
				"namespace": stage.Namespace,
				"depth":     pq.Depth(),
			}).Debug("Stage-specific Promotion queue initialized")
		}
	}
	return nil
}

// syncPromo enqueues Promotion requests to a Stage-specific priority queue. This
// functions assumes the caller has obtained a lock on promoQueuesByStageMu.
func (r *reconciler) syncPromo(
	ctx context.Context,
	promo *kargoapi.Promotion,
) kargoapi.PromotionStatus {
	status := *promo.Status.DeepCopy()

	// Only deal with brand new Promotions
	if promo.Status.Phase != "" {
		return status
	}

	stage := types.NamespacedName{
		Namespace: promo.Namespace,
		Name:      promo.Spec.Stage,
	}

	pq, ok := r.promoQueuesByStage[stage]
	if !ok {
		pq = newPromotionsQueue()
		r.promoQueuesByStage[stage] = pq
	}

	status.Phase = kargoapi.PromotionPhasePending

	// Ignore any errors from this operation. Errors can only occur when you
	// try to push a nil onto the queue and we know we're not doing that.
	pq.Push(promo) // nolint: errcheck

	logging.LoggerFromContext(ctx).WithField("depth", pq.Depth()).
		Infof("pushed Promotion %q to Queue for Stage %q in namespace %q ",
			promo.Name,
			promo.Spec.Stage,
			promo.Namespace,
		)

	return status
}

func (r *reconciler) serializedSync(
	ctx context.Context,
	interval time.Duration,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
		for _, pq := range r.promoQueuesByStage {
			if popped := pq.Pop(); popped != nil {
				promo := popped.(*kargoapi.Promotion) // nolint: forcetypeassert

				logger := logging.LoggerFromContext(ctx).WithFields(log.Fields{
					"promotion": promo.Name,
					"namespace": promo.Namespace,
				})

				// Refresh promo instead of working with something stale
				var err error
				if promo, err = r.getPromo(
					ctx,
					types.NamespacedName{
						Namespace: promo.Namespace,
						Name:      promo.Name,
					},
				); err != nil {
					logger.Error("error finding Promotion")
					continue
				}
				if promo == nil || promo.Status.Phase != kargoapi.PromotionPhasePending {
					continue
				}

				logger = logger.WithFields(log.Fields{
					"stage":   promo.Spec.Stage,
					"freight": promo.Spec.Freight,
				})
				logger.Debug("executing Promotion")

				promoCtx := logging.ContextWithLogger(ctx, logger)

				phase := kargoapi.PromotionPhaseSucceeded
				phaseError := ""

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
						promo.Spec.Stage,
						promo.Namespace,
						promo.Spec.Freight,
					); err != nil {
						phase = kargoapi.PromotionPhaseErrored
						phaseError = err.Error()
						logger.Errorf("error executing Promotion: %s", err)
					}
				}()

				if err = kubeclient.PatchStatus(ctx, r.kargoClient, promo, func(status *kargoapi.PromotionStatus) {
					status.Phase = phase
					status.Error = phaseError
				}); err != nil {
					logger.Errorf("error updating Promotion status: %s", err)
				}

				if promo.Status.Phase == kargoapi.PromotionPhaseSucceeded && err == nil {
					logger.Debug("Promotion succeeded")
				}
			}
		}
	}
}

func (r *reconciler) promote(
	ctx context.Context,
	stageName string,
	stageNamespace string,
	freightID string,
) error {
	logger := logging.LoggerFromContext(ctx)

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

	if currentFreight, ok :=
		stage.Status.History.Top(); ok && currentFreight.ID == freightID {
		logger.Debug("Stage is already in desired Freight")
		return nil
	}

	var targetFreightIndex int
	var targetFreight *kargoapi.Freight
	for i, availableFreight := range stage.Status.AvailableFreight {
		if availableFreight.ID == freightID {
			targetFreightIndex = i
			targetFreight = availableFreight.DeepCopy()
			break
		}
	}
	if targetFreight == nil {
		return errors.Errorf(
			"target Freight %q not found among available Freight of Stage %q "+
				"in namespace %q",
			freightID,
			stageName,
			stageNamespace,
		)
	}

	nextFreight, err := r.promoMechanisms.Promote(ctx, stage, *targetFreight)
	if err != nil {
		return err
	}

	// The assumption is that controller does not process multiple promotions in one stage
	// so we are safe from race conditions and can just update the status
	err = kubeclient.PatchStatus(ctx, r.kargoClient, stage, func(status *kargoapi.StageStatus) {
		status.CurrentFreight = &nextFreight
		status.AvailableFreight[targetFreightIndex] = nextFreight
		status.History.Push(nextFreight)
	})

	return errors.Wrapf(
		err,
		"error updating status of Stage %q in namespace %q",
		stageName,
		stageNamespace,
	)
}

// getPromo returns a pointer to the Promotion resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func (r *reconciler) getPromo(
	ctx context.Context,
	namespacedName types.NamespacedName,
) (*kargoapi.Promotion, error) {
	promo := kargoapi.Promotion{}
	if err := r.kargoClient.Get(ctx, namespacedName, &promo); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			logging.LoggerFromContext(ctx).WithFields(log.Fields{
				"namespace": namespacedName.Namespace,
				"promotion": namespacedName.Name,
			}).Warn("Promotion not found")
			return nil, nil
		}
		return nil, errors.Wrapf(
			err,
			"error getting Promotion %q in namespace %q",
			namespacedName.Name,
			namespacedName.Namespace,
		)
	}
	return &promo, nil
}
