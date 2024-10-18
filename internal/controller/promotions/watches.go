package promotions

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/internal/indexer"
	"github.com/akuity/kargo/internal/logging"
)

// EnqueueHighestPriorityPromotionHandler is an event handler that enqueues the next
// highest priority Promotion for reconciliation when an active Promotion becomes terminal
type EnqueueHighestPriorityPromotionHandler[T any] struct {
	logger      *logging.Logger
	ctx         context.Context
	pqs         *promoQueues
	kargoClient client.Client
}

// Create implements TypedEventHandler.
func (e *EnqueueHighestPriorityPromotionHandler[T]) Create(
	context.Context,
	event.TypedCreateEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Delete implements TypedEventHandler. In case a Running promotion
// becomes deleted, we should enqueue the next one
func (e *EnqueueHighestPriorityPromotionHandler[T]) Delete(
	_ context.Context,
	evt event.TypedDeleteEvent[T],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	if promo, ok := any(evt.Object).(*kargoapi.Promotion); ok {
		stageKey := types.NamespacedName{
			Namespace: promo.Namespace,
			Name:      promo.Spec.Stage,
		}
		e.pqs.conclude(e.ctx, stageKey, promo.Name)
		e.enqueueNext(stageKey, wq)
	}
}

// Generic implements TypedEventHandler.
func (e *EnqueueHighestPriorityPromotionHandler[T]) Generic(
	context.Context,
	event.TypedGenericEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Update implements TypedEventHandler. This should only be called with
// a promo that transitioned from non-terminal to terminal.
func (e *EnqueueHighestPriorityPromotionHandler[T]) Update(
	_ context.Context,
	evt event.TypedUpdateEvent[T],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	promo := any(evt.ObjectNew).(*kargoapi.Promotion) // nolint: forcetypeassert
	if promo == nil {
		e.logger.Error(
			nil, "Update event has no new object to update",
			"event", evt,
		)
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
func (e *EnqueueHighestPriorityPromotionHandler[T]) enqueueNext(
	stageKey types.NamespacedName,
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
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
			e.logger.Error(
				err, "Failed to get next highest priority Promotion for enqueue",
				"firstKey", firstKey,
			)
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
		e.logger.Debug(
			"enqueued promo",
			"promotion", promo.Name,
			"namespace", promo.Namespace,
			"stage", promo.Spec.Stage,
		)
		return
	}
}

// UpdatedArgoCDAppHandler is an event handler that enqueues Promotions for
// reconciliation when an associated ArgoCD Application is updated.
type UpdatedArgoCDAppHandler[T any] struct {
	kargoClient   client.Client
	shardSelector labels.Selector
}

// Create implements TypedEventHandler.
func (u *UpdatedArgoCDAppHandler[T]) Create(
	context.Context,
	event.TypedCreateEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Delete implements TypedEventHandler.
func (u *UpdatedArgoCDAppHandler[T]) Delete(
	context.Context,
	event.TypedDeleteEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Generic implements TypedEventHandler.
func (u *UpdatedArgoCDAppHandler[T]) Generic(
	context.Context,
	event.TypedGenericEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Update implements TypedEventHandler.
func (u *UpdatedArgoCDAppHandler[T]) Update(
	ctx context.Context,
	e event.TypedUpdateEvent[T],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	logger := logging.LoggerFromContext(ctx)

	oldApp := any(e.ObjectOld).(*argocd.Application) // nolint: forcetypeassert
	newApp := any(e.ObjectNew).(*argocd.Application) // nolint: forcetypeassert
	if newApp == nil || oldApp == nil {
		logger.Error(
			nil, "Update event has no new or old object to update",
			"event", e,
		)
		return
	}

	promotions := &kargoapi.PromotionList{}
	if err := u.kargoClient.List(
		ctx,
		promotions,
		&client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(
				indexer.RunningPromotionsByArgoCDApplicationsIndexField,
				fmt.Sprintf("%s:%s", newApp.Namespace, newApp.Name),
			),
			LabelSelector: u.shardSelector,
		},
	); err != nil {
		logger.Error(
			err, "error listing Promotions for Application",
			"app", newApp.Name,
			"namespace", newApp.Namespace,
		)
		return
	}

	for _, promotion := range promotions.Items {
		wq.Add(
			reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: promotion.Namespace,
					Name:      promotion.Name,
				},
			},
		)
		logger.Debug(
			"enqueued Promotion for reconciliation",
			"namespace", promotion.Namespace,
			"promotion", promotion.Name,
			"app", newApp.Name,
		)
	}
}

// PromotionAcknowledgedByStageHandler is an event handler that enqueues a
// Promotion for reconciliation when it has been acknowledged by the Stage\
// it is for.
type PromotionAcknowledgedByStageHandler[T any] struct{}

// Create implements TypedEventHandler.
func (u *PromotionAcknowledgedByStageHandler[T]) Create(
	context.Context,
	event.TypedCreateEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Delete implements TypedEventHandler.
func (u *PromotionAcknowledgedByStageHandler[T]) Delete(
	context.Context,
	event.TypedDeleteEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Generic implements TypedEventHandler.
func (u *PromotionAcknowledgedByStageHandler[T]) Generic(
	context.Context,
	event.TypedGenericEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Update implements TypedEventHandler.
func (u *PromotionAcknowledgedByStageHandler[T]) Update(
	ctx context.Context,
	e event.TypedUpdateEvent[T],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	logger := logging.LoggerFromContext(ctx)

	oldStage := any(e.ObjectOld).(*kargoapi.Stage) // nolint: forcetypeassert
	newStage := any(e.ObjectNew).(*kargoapi.Stage) // nolint: forcetypeassert
	if newStage == nil || oldStage == nil {
		logger.Error(
			nil, "Update event has no new or old object",
			"event", e,
		)
		return
	}

	if newStage.Status.CurrentPromotion == nil {
		return
	}

	if oldStage.Status.CurrentPromotion == nil ||
		oldStage.Status.CurrentPromotion.Name == newStage.Status.CurrentPromotion.Name {
		wq.Add(reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: newStage.Namespace,
				Name:      newStage.Status.CurrentPromotion.Name,
			},
		})
		logger.Debug(
			"enqueued Promotion for reconciliation",
			"namespace", newStage.Namespace,
			"promotion", newStage.Status.CurrentPromotion.Name,
			"stage", newStage.Name,
		)
	}
}
