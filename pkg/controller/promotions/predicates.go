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

// ArgoCDAppHealthChanged is a predicate that fires when an ArgoCD
// Application's health status changes (e.g. Progressing → Healthy). This
// enables the Promotion reconciler to react promptly to health transitions
// instead of waiting for a polling interval.
type ArgoCDAppHealthChanged[T any] struct {
	logger *logging.Logger
}

func (p ArgoCDAppHealthChanged[T]) Create(event.TypedCreateEvent[T]) bool {
	return false
}

func (p ArgoCDAppHealthChanged[T]) Update(e event.TypedUpdateEvent[T]) bool {
	oldApp := any(e.ObjectOld).(*argocd.Application) // nolint: forcetypeassert
	newApp := any(e.ObjectNew).(*argocd.Application) // nolint: forcetypeassert
	if oldApp == nil || newApp == nil {
		p.logger.Error(
			nil, "Update event has no new or old object",
			"event", e,
		)
		return false
	}
	return newApp.Status.Health.Status != oldApp.Status.Health.Status
}

func (p ArgoCDAppHealthChanged[T]) Delete(event.TypedDeleteEvent[T]) bool {
	return false
}

func (p ArgoCDAppHealthChanged[T]) Generic(event.TypedGenericEvent[T]) bool {
	return false
}

// ArgoCDAppSyncChanged is a predicate that fires when an ArgoCD Application's
// sync status changes (e.g. OutOfSync → Synced). This enables the Promotion
// reconciler to react promptly to sync transitions instead of waiting for a
// polling interval.
type ArgoCDAppSyncChanged[T any] struct {
	logger *logging.Logger
}

func (p ArgoCDAppSyncChanged[T]) Create(event.TypedCreateEvent[T]) bool {
	return false
}

func (p ArgoCDAppSyncChanged[T]) Update(e event.TypedUpdateEvent[T]) bool {
	oldApp := any(e.ObjectOld).(*argocd.Application) // nolint: forcetypeassert
	newApp := any(e.ObjectNew).(*argocd.Application) // nolint: forcetypeassert
	if oldApp == nil || newApp == nil {
		p.logger.Error(
			nil, "Update event has no new or old object",
			"event", e,
		)
		return false
	}
	return newApp.Status.Sync.Status != oldApp.Status.Sync.Status
}

func (p ArgoCDAppSyncChanged[T]) Delete(event.TypedDeleteEvent[T]) bool {
	return false
}

func (p ArgoCDAppSyncChanged[T]) Generic(event.TypedGenericEvent[T]) bool {
	return false
}

// ArgoCDAppReconciledAfterOperation is a predicate that fires when an Argo CD
// Application's reconciledAt advances from a stale value (nil or before
// finishedAt) to a newer value. This is the signal that Argo CD has
// re-assessed health after a completed operation, making the health status
// trustworthy again. It enables the Promotion reconciler to react promptly
// when a hard refresh completes rather than waiting for the next polling
// interval.
type ArgoCDAppReconciledAfterOperation[T any] struct {
	logger *logging.Logger
}

func (p ArgoCDAppReconciledAfterOperation[T]) Create(event.TypedCreateEvent[T]) bool {
	return false
}

func (p ArgoCDAppReconciledAfterOperation[T]) Update(e event.TypedUpdateEvent[T]) bool {
	oldApp := any(e.ObjectOld).(*argocd.Application) // nolint: forcetypeassert
	newApp := any(e.ObjectNew).(*argocd.Application) // nolint: forcetypeassert
	if oldApp == nil || newApp == nil {
		p.logger.Error(
			nil, "Update event has no new or old object",
			"event", e,
		)
		return false
	}

	// Only relevant when there is a completed operation with a known finish time.
	if newApp.Status.OperationState == nil || newApp.Status.OperationState.FinishedAt == nil {
		return false
	}
	finishedAt := newApp.Status.OperationState.FinishedAt

	// reconciledAt must have changed.
	if oldApp.Status.ReconciledAt.Equal(newApp.Status.ReconciledAt) {
		return false
	}

	// Only fire if the old reconciledAt was stale (nil or before finishedAt).
	// Once reconciledAt is already trusted (>= finishedAt), subsequent
	// advances are routine Argo CD refreshes that do not need to wake the
	// Promotion reconciler.
	return oldApp.Status.ReconciledAt == nil ||
		oldApp.Status.ReconciledAt.Before(finishedAt)
}

func (p ArgoCDAppReconciledAfterOperation[T]) Delete(event.TypedDeleteEvent[T]) bool {
	return false
}

func (p ArgoCDAppReconciledAfterOperation[T]) Generic(event.TypedGenericEvent[T]) bool {
	return false
}
