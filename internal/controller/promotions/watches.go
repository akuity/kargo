package promotions

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// EnqueueHighestPriorityPromotionHandler is an event handler that enqueues the next
// highest priority Promotion for reconciliation when an active Promotion becomes terminal
type EnqueueHighestPriorityPromotionHandler struct {
	logger      logr.Logger
	ctx         context.Context
	pqs         *promoQueues
	kargoClient client.Client
}

// Create implements EventHandler.
func (e *EnqueueHighestPriorityPromotionHandler) Create(
	context.Context,
	event.CreateEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Delete implements EventHandler. In case a Running promotion
// becomes deleted, we should enqueue the next one
func (e *EnqueueHighestPriorityPromotionHandler) Delete(
	_ context.Context,
	evt event.DeleteEvent,
	wq workqueue.RateLimitingInterface,
) {
	if promo, ok := evt.Object.(*kargoapi.Promotion); ok {
		stageKey := types.NamespacedName{
			Namespace: promo.Namespace,
			Name:      promo.Spec.Stage,
		}
		e.pqs.conclude(e.ctx, stageKey, promo.Name)
		e.enqueueNext(stageKey, wq)
	}
}

// Generic implements EventHandler.
func (e *EnqueueHighestPriorityPromotionHandler) Generic(
	context.Context,
	event.GenericEvent,
	workqueue.RateLimitingInterface,
) {
	// No-op
}

// Update implements EventHandler. This should only be called with
// a promo that transitioned from non-terminal to terminal.
func (e *EnqueueHighestPriorityPromotionHandler) Update(
	_ context.Context,
	evt event.UpdateEvent,
	wq workqueue.RateLimitingInterface,
) {
	if evt.ObjectNew == nil {
		e.logger.Error(nil, "Update event has no new object to update", "object", evt)
		return
	}
	promo, ok := evt.ObjectNew.(*kargoapi.Promotion)
	if !ok {
		e.logger.Error(nil, "Failed to convert new Promotion", "promotion", evt.ObjectNew)
		return
	}
	if promo.Status.Phase.IsTerminal() {
		stageKey := types.NamespacedName{
			Namespace: promo.Namespace,
			Name:      promo.Spec.Stage,
		}
		// This promo just went terminal. Deactivate it and enqueue
		// the next highest priority promo for reconciliation
		e.pqs.conclude(e.ctx, stageKey, promo.Name)
		e.enqueueNext(stageKey, wq)
	}
}

// enqueueNext enqueues the next highest priority promotion for reconciliation to the workqueue.
// Also discards pending promotions in the queue that no longer exist
func (e *EnqueueHighestPriorityPromotionHandler) enqueueNext(
	stageKey types.NamespacedName,
	wq workqueue.RateLimitingInterface,
) {
	e.pqs.promoQueuesByStageMu.RLock()
	defer e.pqs.promoQueuesByStageMu.RUnlock()
	if e.pqs.activePromoByStage[stageKey] != "" {
		// there's already an active promotion. don't need to enqueue the next one
		return
	}
	pq, ok := e.pqs.pendingPromoQueuesByStage[stageKey]
	if !ok {
		return
	}

	// NOTE: at first glance, this for loop appears to be expensive to do while holding
	// the pqs mutex. But it isn't as bad as it looks, since we count on the fact that
	// GetPromotion calls pull from the informer cache and do not involve an HTTP call.
	// and in the common case, we only do a single iteration
	for {
		first := pq.Peek()
		if first == nil {
			// pending queue is empty
			return
		}
		// Check if promo exists, and enqueue it if it does
		firstKey := types.NamespacedName{Namespace: first.GetNamespace(), Name: first.GetName()}
		promo, err := kargoapi.GetPromotion(e.ctx, e.kargoClient, firstKey)
		if err != nil {
			e.logger.Error(err, "Failed to get next highest priority Promotion for enqueue", "promotion", firstKey)
			return
		}
		if promo == nil || promo.Status.Phase.IsTerminal() {
			// Found a promotion in the pending queue that no longer exists
			// or terminal. Pop it and loop to the next item in the queue
			_ = pq.Pop()
			continue
		}
		wq.AddRateLimited(
			reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: promo.Namespace,
					Name:      promo.Name,
				},
			},
		)
		e.logger.V(1).Info(
			"enqueued promo",
			"promotion", promo.Name,
			"namespace", promo.Namespace,
			"stage", promo.Spec.Stage,
		)
		return
	}
}
