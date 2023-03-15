package controller

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	api "github.com/akuityio/kargo/api/v1alpha1"
	"github.com/akuityio/kargo/internal/config"
	"github.com/akuityio/kargo/internal/controller/runtime"
)

// promotionReconciler reconciles Promotion resources.
type promotionReconciler struct {
	client             client.Client
	promoQueuesByEnv   map[types.NamespacedName]runtime.PriorityQueue
	promoQueuesByEnvMu sync.Mutex
	logger             *log.Logger
	initializeOnce     sync.Once
}

// SetupPromotionReconcilerWithManager initializes a reconciler for
// Promotion resources and registers it with the provided Manager.
func SetupPromotionReconcilerWithManager(
	ctx context.Context,
	config config.ControllerConfig,
	mgr manager.Manager,
) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Promotion{}).
		WithEventFilter(predicate.Funcs{}).
		Complete(
			newPromotionReconciler(config, mgr.GetClient()),
		)
}

func newPromotionReconciler(
	config config.ControllerConfig,
	client client.Client,
) *promotionReconciler {
	logger := log.New()
	logger.SetLevel(config.LogLevel)
	return &promotionReconciler{
		client:           client,
		promoQueuesByEnv: map[types.NamespacedName]runtime.PriorityQueue{},
		logger:           logger,
	}
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
func (p *promotionReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	// We count all of Reconcile() as a critical section of code to ensure we
	// don't start reconciling a second Promotion before lazy initialization
	// completes upon reconciliation of the FIRST promotion.
	p.promoQueuesByEnvMu.Lock()
	defer p.promoQueuesByEnvMu.Unlock()

	result := ctrl.Result{}

	// Note that initialization occurs here because we basically know that the
	// controller runtime client's cache is ready at this point. We cannot attempt
	// to list Promotions prior to that point.
	var err error
	p.initializeOnce.Do(func() {
		if err = p.initializeQueues(ctx); err == nil {
			p.logger.Debug(
				"initialized Environment-specific Promotion queues from list of " +
					"existing Promotions",
			)
		}
		// TODO: Do not hardcode this interval
		go p.serializedSync(ctx, 10*time.Second)
	})
	if err != nil {
		return result, errors.Wrap(err, "error initializing Promotion queues")
	}

	// Find the Promotion
	promo, err := p.getPromo(ctx, req.NamespacedName)
	if err != nil {
		return result, err
	}
	if promo == nil {
		// Ignore if not found. This can happen if the Promotion was deleted after
		// the current reconciliation request was issued.
		return result, nil
	}

	p.logger.WithFields(log.Fields{
		"namespace": req.NamespacedName.Namespace,
		"name":      req.NamespacedName.Name,
	}).Debug("reconciling Promotion")

	promo.Status = p.sync(ctx, promo)
	p.updateStatus(ctx, promo)

	return result, err
}

// initializeQueues lists all Promotions and adds them to relevant priority
// queues. This is intended to be invoked ONCE and the caller MUST ensure that.
// It is also assumed that the caller has already obtained a lock on
// promoQueuesByEnvMu.
func (p *promotionReconciler) initializeQueues(ctx context.Context) error {
	promos := api.PromotionList{}
	if err := p.client.List(ctx, &promos); err != nil {
		return errors.Wrap(err, "error listing promotions")
	}
	for _, promo := range promos.Items {
		switch promo.Status.Phase {
		case api.PromotionPhaseComplete, api.PromotionPhaseFailed:
			continue
		case "":
			promo.Status.Phase = api.PromotionPhasePending
			if err := p.client.Status().Update(ctx, &promo); err != nil {
				return errors.Wrapf(
					err,
					"error updating status of Promotion %q in namespace %q",
					promo.Name,
					promo.Namespace,
				)
			}
		}
		env := types.NamespacedName{
			Namespace: promo.Namespace,
			Name:      promo.Spec.Environment,
		}
		pq, ok := p.promoQueuesByEnv[env]
		if !ok {
			pq = newPromotionsQueue()
			p.promoQueuesByEnv[env] = pq
		}
		// The only error that can occur here happens when you push a nil and we
		// know we're not doing that.
		pq.Push(&promo) // nolint: errcheck
		p.logger.WithFields(log.Fields{
			"name":        promo.Name,
			"namespace":   promo.Namespace,
			"environment": promo.Spec.Environment,
			"phase":       promo.Status.Phase,
		}).Debug("pushed Promotion onto Environment-specific Promotion queue")
	}
	if p.logger.IsLevelEnabled(log.DebugLevel) {
		for env, pq := range p.promoQueuesByEnv {
			p.logger.WithFields(log.Fields{
				"environment": env.Name,
				"namespace":   env.Namespace,
				"depth":       pq.Depth(),
			}).Debug("Environment-specific Promotion queue initialized")
		}
	}
	return nil
}

// sync enqueues Promotion requests to an Environment-specific priority queue.
// This functions assumes the caller has obtained a lock on promoQueuesByEnvMu.
func (p *promotionReconciler) sync(
	ctx context.Context,
	promo *api.Promotion,
) api.PromotionStatus {
	status := *promo.Status.DeepCopy()

	// Only deal with brand new Promotions
	if promo.Status.Phase != "" {
		return status
	}

	promo.Status.Phase = api.PromotionPhasePending

	env := types.NamespacedName{
		Namespace: promo.Namespace,
		Name:      promo.Spec.Environment,
	}

	pq, ok := p.promoQueuesByEnv[env]
	if !ok {
		pq = newPromotionsQueue()
		p.promoQueuesByEnv[env] = pq
	}

	status.Phase = api.PromotionPhasePending

	// Ignore any errors from this operation. Errors can only occur when you
	// try to push a nil onto the queue and we know we're not doing that.
	pq.Push(promo) // nolint: errcheck

	p.logger.WithField("depth", pq.Depth()).
		Infof("pushed Promotion %q to Queue for Environment %q in namespace %q ",
			promo.Name,
			promo.Spec.Environment,
			promo.Namespace,
		)

	return status
}

func (p *promotionReconciler) serializedSync(
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
		for _, pq := range p.promoQueuesByEnv {
			if popped := pq.Pop(); popped != nil {
				promo := popped.(*api.Promotion)

				promoLogger := p.logger.WithFields(log.Fields{
					"name":      promo.Name,
					"namespace": promo.Namespace,
				})

				// Refresh promo instead of working with something stale
				var err error
				if promo, err = p.getPromo(
					ctx,
					types.NamespacedName{
						Namespace: promo.Namespace,
						Name:      promo.Name,
					},
				); err != nil {
					promoLogger.Error("error finding Promotion")
				}

				// TODO: Actual promotion logic goes here

				promo.Status.Phase = api.PromotionPhaseComplete
				p.updateStatus(ctx, promo)
				promoLogger.WithFields(log.Fields{
					"environment": promo.Spec.Environment,
					"state":       promo.Spec.State,
				}).Debug("handled Promotion")
			}
		}
	}
}

// getPromo returns a pointer to the Promotion resource specified by the
// namespacedName argument. If no such resource is found, nil is returned
// instead.
func (p *promotionReconciler) getPromo(
	ctx context.Context,
	namespacedName types.NamespacedName,
) (*api.Promotion, error) {
	promo := api.Promotion{}
	if err := p.client.Get(ctx, namespacedName, &promo); err != nil {
		if err = client.IgnoreNotFound(err); err == nil {
			p.logger.WithFields(log.Fields{
				"namespace": namespacedName.Namespace,
				"name":      namespacedName.Name,
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

// updateStatus updates the status subresource of the provided Promotion.
func (p *promotionReconciler) updateStatus(
	ctx context.Context,
	promo *api.Promotion,
) {
	if err := p.client.Status().Update(ctx, promo); err != nil {
		p.logger.WithFields(log.Fields{
			"namespace": promo.Namespace,
			"name":      promo.Name,
		}).Errorf("error updating Promotion status: %s", err)
	}
}
