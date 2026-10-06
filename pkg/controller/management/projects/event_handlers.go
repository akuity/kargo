package projects

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/logging"
)

// projectWarehouseHealthEnqueuer enqueues a Project for reconciliation when the
// health condition of a Warehouse within that Project changes.
type projectWarehouseHealthEnqueuer[T any] struct{}

// Create implements TypedEventHandler.
func (e *projectWarehouseHealthEnqueuer[T]) Create(
	context.Context,
	event.TypedCreateEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Delete implements TypedEventHandler.
func (e *projectWarehouseHealthEnqueuer[T]) Delete(
	context.Context,
	event.TypedDeleteEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Generic implements TypedEventHandler.
func (e *projectWarehouseHealthEnqueuer[T]) Generic(
	context.Context,
	event.TypedGenericEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Update implements TypedEventHandler.
func (e *projectWarehouseHealthEnqueuer[T]) Update(
	ctx context.Context,
	evt event.TypedUpdateEvent[T],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	logger := logging.LoggerFromContext(ctx)

	oldWarehouse, ok := any(evt.ObjectOld).(*kargoapi.Warehouse)
	if !ok {
		return
	}
	newWarehouse, ok := any(evt.ObjectNew).(*kargoapi.Warehouse)
	if !ok {
		return
	}

	if oldWarehouse == nil || newWarehouse == nil {
		logger.Error(
			nil, "Update event has no old or new object to update",
			"event", evt,
		)
		return
	}

	oldCond := conditions.Get(&oldWarehouse.Status, kargoapi.ConditionTypeHealthy)
	newCond := conditions.Get(&newWarehouse.Status, kargoapi.ConditionTypeHealthy)
	switch {
	case oldCond == nil && newCond == nil:
		return
	case oldCond == nil || newCond == nil:
		fallthrough
	case oldCond.Status != newCond.Status:
		logger.Info("Warehouse health changed, enqueueing Project")
		wq.Add(reconcile.Request{
			NamespacedName: types.NamespacedName{Name: newWarehouse.Namespace},
		})
	}
}

// projectStageHealthEnqueuer enqueues a Project for reconciliation when the
// health condition of a Stage within that Project changes, or when the
// PromotionRequest the Stage reports as current or last changes. The latter
// marks a round of promotion to the Stage's Targets starting or ending, which
// is when the Project's Target stats are recomputed. Those stats are a
// high-level overview, so they deliberately do not track each child Promotion
// finishing mid-round, or Targets appearing and disappearing; watching those
// would multiply Project reconciles for little benefit.
type projectStageHealthEnqueuer[T any] struct{}

// Create implements TypedEventHandler.
func (e *projectStageHealthEnqueuer[T]) Create(
	context.Context,
	event.TypedCreateEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Delete implements TypedEventHandler.
func (e *projectStageHealthEnqueuer[T]) Delete(
	context.Context,
	event.TypedDeleteEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Generic implements TypedEventHandler.
func (e *projectStageHealthEnqueuer[T]) Generic(
	context.Context,
	event.TypedGenericEvent[T],
	workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	// No-op
}

// Update implements TypedEventHandler.
func (e *projectStageHealthEnqueuer[T]) Update(
	ctx context.Context,
	evt event.TypedUpdateEvent[T],
	wq workqueue.TypedRateLimitingInterface[reconcile.Request],
) {
	logger := logging.LoggerFromContext(ctx)

	oldStage, ok := any(evt.ObjectOld).(*kargoapi.Stage)
	if !ok {
		return
	}
	newStage, ok := any(evt.ObjectNew).(*kargoapi.Stage)
	if !ok {
		return
	}

	if oldStage == nil || newStage == nil {
		logger.Error(
			nil, "Update event has no old or new object to update",
			"event", evt,
		)
		return
	}

	if promotionRequestRefName(oldStage.Status.CurrentPromotionRequest) !=
		promotionRequestRefName(newStage.Status.CurrentPromotionRequest) ||
		promotionRequestRefName(oldStage.Status.LastPromotionRequest) !=
			promotionRequestRefName(newStage.Status.LastPromotionRequest) {
		logger.Info("Stage's PromotionRequest changed, enqueueing Project")
		wq.Add(reconcile.Request{
			NamespacedName: types.NamespacedName{Name: newStage.Namespace},
		})
		return
	}

	oldCond := conditions.Get(&oldStage.Status, kargoapi.ConditionTypeHealthy)
	newCond := conditions.Get(&newStage.Status, kargoapi.ConditionTypeHealthy)
	switch {
	case oldCond == nil && newCond == nil:
		return
	case oldCond == nil || newCond == nil:
		fallthrough
	case oldCond.Status != newCond.Status:
		logger.Info("Stage health changed, enqueueing Project")
		wq.Add(reconcile.Request{
			NamespacedName: types.NamespacedName{Name: oldStage.Namespace},
		})
	}
}

// promotionRequestRefName returns the name a Stage's PromotionRequest reference
// carries, or the empty string when there is no reference.
func promotionRequestRefName(ref *kargoapi.PromotionRequestReference) string {
	if ref == nil {
		return ""
	}
	return ref.Name
}
