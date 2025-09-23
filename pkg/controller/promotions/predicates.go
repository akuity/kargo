package promotions

import (
	"sigs.k8s.io/controller-runtime/pkg/event"

	argocd "github.com/akuity/kargo/pkg/controller/argocd/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

// ArgoCDAppOperationCompleted is a predicate that filters out ArgoCD Application
// Update events where the operation has completed. This is useful for triggering
// a reconciliation of a Promotion only when an ArgoCD Application operation has
// finished.
type ArgoCDAppOperationCompleted[T any] struct {
	logger *logging.Logger
}

func (p ArgoCDAppOperationCompleted[T]) Create(event.TypedCreateEvent[T]) bool {
	return false
}

func (p ArgoCDAppOperationCompleted[T]) Update(e event.TypedUpdateEvent[T]) bool {
	oldApp := any(e.ObjectOld).(*argocd.Application) // nolint: forcetypeassert
	if oldApp == nil {
		p.logger.Error(
			nil, "Update event has no old object to update",
			"event", e,
		)
		return false
	}
	newApp := any(e.ObjectNew).(*argocd.Application) // nolint: forcetypeassert
	if newApp == nil {
		p.logger.Error(
			nil, "Update event has no new object for update",
			"event", e,
		)
		return false
	}

	if newApp.Status.OperationState == nil {
		// No operation state to compare against.
		return false
	}

	newOperationCompleted := newApp.Status.OperationState.Phase.Completed()
	oldOperationCompleted := oldApp.Status.OperationState != nil && oldApp.Status.OperationState.Phase.Completed()

	return newOperationCompleted && !oldOperationCompleted
}

func (p ArgoCDAppOperationCompleted[T]) Delete(event.TypedDeleteEvent[T]) bool {
	return false
}

func (p ArgoCDAppOperationCompleted[T]) Generic(event.TypedGenericEvent[T]) bool {
	return false
}
