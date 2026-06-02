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

	// Collect the running Promotions associated with this Application, deduped by
	// key in case a Promotion is matched by both lookups below.
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
			"app", newApp.Name,
		)
	}

	// Promotions that reference this Application by name are found via the
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

	// Promotions that selected this Application by label selector are not
	// captured by the name-based index. The Application's authorized-stage
	// annotation names the Stage(s) authorized to manage it, so enqueue the
	// running Promotions for those Stages. This is a correct superset: a
	// spuriously enqueued Promotion simply results in an idempotent re-check.
	annotation, ok := newApp.Annotations[kargoapi.AnnotationKeyAuthorizedStage]
	if !ok {
		return
	}
	authorizedStages, err := kargoapi.AuthorizedStages(annotation)
	if err != nil {
		logger.Error(
			err, "error parsing authorized Stages for Application",
			"app", newApp.Name,
			"namespace", newApp.Namespace,
		)
		return
	}
	for _, stage := range authorizedStages {
		stagePromotions := &kargoapi.PromotionList{}
		if err := u.kargoClient.List(
			ctx,
			stagePromotions,
			client.InNamespace(stage.Namespace),
			client.MatchingFields{
				// Note: This index only includes running Promotions assigned to
				// this shard.
				indexer.RunningPromotionsByStageField: stage.Name,
			},
		); err != nil {
			logger.Error(
				err, "error listing Promotions for Stage authorized by Application",
				"app", newApp.Name,
				"namespace", stage.Namespace,
				"stage", stage.Name,
			)
			continue
		}
		for _, promotion := range stagePromotions.Items {
			enqueue(promotion)
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
