package projects

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/conditions"
	"github.com/akuity/kargo/internal/logging"
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
// health condition of a Stage within that Project changes.
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

	oldCond := conditions.Get(&oldStage.Status, kargoapi.ConditionTypeHealthy)
	newCond := conditions.Get(&newStage.Status, kargoapi.ConditionTypeHealthy)
	switch {
	case oldCond == nil && newCond == nil:
		return
	case oldCond == nil || newCond == nil:
		fallthrough
	case oldCond.Status != newCond.Status:
		logger.Info("Warehouse health changed, enqueueing Project")
		wq.Add(reconcile.Request{
			NamespacedName: types.NamespacedName{Name: oldStage.Namespace},
		})
	}
}
