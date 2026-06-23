package promotions

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller"
	argocd "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/logging"
)

// UpdatedArgoCDAppHandler is an event handler that enqueues Promotions for
// reconciliation when an associated ArgoCD Application is updated.
type UpdatedArgoCDAppHandler[T any] struct {
	kargoClient client.Client
}

// Create implements TypedEventHandler.
func (u *UpdatedArgoCDAppHandler[T]) Create(
	ctx context.Context,
	e event.TypedCreateEvent[T],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	logger := logging.LoggerFromContext(ctx)
	app := any(e.Object).(*argocd.Application) // nolint: forcetypeassert
	if app == nil {
		logger.Error(nil, "Create event has no object", "event", e)
		return
	}
	// A newly-created Application can enter a label selector's match set. Name-
	// based targeting is unaffected by creation (such a Promotion already
	// references the Application by name and will be reconciled when the
	// Application is next updated), so only selector-based Promotions are woken.
	enqueued, enqueue := newEnqueuer(logger, wq, app.Name)
	u.enqueueSelectorMatches(ctx, enqueued, enqueue, app)
}

// Delete implements TypedEventHandler.
func (u *UpdatedArgoCDAppHandler[T]) Delete(
	ctx context.Context,
	e event.TypedDeleteEvent[T],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	logger := logging.LoggerFromContext(ctx)
	app := any(e.Object).(*argocd.Application) // nolint: forcetypeassert
	if app == nil {
		logger.Error(nil, "Delete event has no object", "event", e)
		return
	}
	// A deleted Application leaves a label selector's match set. The deleted
	// object still carries its labels, so we can evaluate selectors against it.
	enqueued, enqueue := newEnqueuer(logger, wq, app.Name)
	u.enqueueSelectorMatches(ctx, enqueued, enqueue, app)
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

	// Collect the running Promotions to enqueue, deduped by key in case a
	// Promotion is matched by both the name- and selector-based lookups below.
	enqueued, enqueue := newEnqueuer(logger, wq, newApp.Name)

	// Promotions that target this Application by name are found directly via the
	// name-based index.
	promotions := &kargoapi.PromotionList{}
	if err := u.kargoClient.List(
		ctx,
		promotions,
		&client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(
				// Note: This index only includes Promotions assigned to this shard.
				indexer.RunningPromotionsByArgoCDApplicationsField,
				fmt.Sprintf("%s:%s", newApp.Namespace, newApp.Name),
			),
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
		enqueue(promotion)
	}

	// Promotions that target this Application by label selector cannot be found
	// via the name-based index, because a selector's match set depends on the
	// Application's labels. We test both the old and new label sets so a
	// Promotion is still woken when an Application's labels change such that it
	// enters or stops matching the selector.
	u.enqueueSelectorMatches(ctx, enqueued, enqueue, newApp, oldApp)
}

// newEnqueuer returns a dedupe set and an enqueue function that adds a Promotion
// to the workqueue at most once, logging the Application that triggered it.
func newEnqueuer(
	logger *logging.Logger,
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
	appName string,
) (map[types.NamespacedName]struct{}, func(kargoapi.Promotion)) {
	enqueued := make(map[types.NamespacedName]struct{})
	enqueue := func(promo kargoapi.Promotion) {
		key := types.NamespacedName{Namespace: promo.Namespace, Name: promo.Name}
		if _, ok := enqueued[key]; ok {
			return
		}
		enqueued[key] = struct{}{}
		wq.Add(reconcile.Request{NamespacedName: key})
		logger.Debug(
			"enqueued Promotion for reconciliation",
			"namespace", promo.Namespace,
			"promotion", promo.Name,
			"app", appName,
		)
	}
	return enqueued, enqueue
}

// enqueueSelectorMatches enqueues every running, selector-based Promotion in
// this shard whose argocd-update/argocd-wait selectors match any of the given
// Applications. The candidate set is narrowed via a coarse index
// (RunningPromotionsByArgoCDSelectorsField) before each selector is evaluated
// forward against the Applications. Passing multiple Applications (e.g. an
// Update event's old and new states) wakes a Promotion both when an Application
// enters and when it leaves a selector's match set.
func (u *UpdatedArgoCDAppHandler[T]) enqueueSelectorMatches(
	ctx context.Context,
	enqueued map[types.NamespacedName]struct{},
	enqueue func(kargoapi.Promotion),
	apps ...*argocd.Application,
) {
	logger := logging.LoggerFromContext(ctx)
	selectorPromotions := &kargoapi.PromotionList{}
	if err := u.kargoClient.List(
		ctx,
		selectorPromotions,
		&client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(
				// Note: This index only includes Promotions assigned to this shard.
				indexer.RunningPromotionsByArgoCDSelectorsField,
				indexer.RunningPromotionsByArgoCDSelectorsValue,
			),
		},
	); err != nil {
		logger.Error(err, "error listing selector-based Promotions for Application")
		return
	}
	for i := range selectorPromotions.Items {
		promotion := selectorPromotions.Items[i]
		if _, ok := enqueued[types.NamespacedName{
			Namespace: promotion.Namespace,
			Name:      promotion.Name,
		}]; ok {
			continue
		}
		for _, app := range apps {
			if app == nil {
				continue
			}
			if promotionSelectorsMatchApp(ctx, u.kargoClient, &promotion, app.Namespace, app.Labels) {
				enqueue(promotion)
				break
			}
		}
	}
}

// NewPromotionAcknowledgedByStageHandler creates a new
// PromotionAcknowledgedByStageHandler with the given shard predicate.
func NewPromotionAcknowledgedByStageHandler[T any](
	shardPredicate controller.ResponsibleFor[kargoapi.Stage],
) *PromotionAcknowledgedByStageHandler[T] {
	return &PromotionAcknowledgedByStageHandler[T]{
		shardPredicate: shardPredicate,
	}
}

// PromotionAcknowledgedByStageHandler is an event handler that enqueues a
// Promotion for reconciliation when it has been acknowledged by the Stage\
// it is for.
type PromotionAcknowledgedByStageHandler[T any] struct {
	shardPredicate controller.ResponsibleFor[kargoapi.Stage]
}

// Create implements TypedEventHandler.
func (p *PromotionAcknowledgedByStageHandler[T]) Create(
	context.Context,
	event.TypedCreateEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Delete implements TypedEventHandler.
func (p *PromotionAcknowledgedByStageHandler[T]) Delete(
	context.Context,
	event.TypedDeleteEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Generic implements TypedEventHandler.
func (p *PromotionAcknowledgedByStageHandler[T]) Generic(
	context.Context,
	event.TypedGenericEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Update implements TypedEventHandler.
func (p *PromotionAcknowledgedByStageHandler[T]) Update(
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

	// When an event handler places work on the work queue, it bypasses the event
	// filters the reconciler may be using on its watches, so we want to be sure
	// here that we do not enqueue a Stage's current Promotion for
	// reconciliation if the Stage isn't handled by this shard. (The Promotions
	// reconciler' Reconcile() method will ultimately ignore any such Promotion
	// anyway, so really this is just an optimization.)
	if !p.shardPredicate.IsResponsible(newStage) {
		return
	}

	if newStage.Status.CurrentPromotion == nil {
		return
	}

	if oldStage.Status.CurrentPromotion == nil ||
		oldStage.Status.CurrentPromotion.Name != newStage.Status.CurrentPromotion.Name {
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
