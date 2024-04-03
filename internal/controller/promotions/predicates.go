package promotions

import (
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/controller-runtime/pkg/event"

	argocd "github.com/akuity/kargo/internal/controller/argocd/api/v1alpha1"
)

// ArgoCDAppOperationCompleted is a predicate that filters out ArgoCD Application
// Update events where the operation has completed. This is useful for triggering
// a reconciliation of a Promotion only when an ArgoCD Application operation has
// finished.
type ArgoCDAppOperationCompleted struct {
	logger log.FieldLogger
}

func (p ArgoCDAppOperationCompleted) Create(event.CreateEvent) bool {
	return false
}

func (p ArgoCDAppOperationCompleted) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		p.logger.Errorf("Update event has no old object to update: %v", e)
		return false
	}
	if e.ObjectNew == nil {
		p.logger.Errorf("Update event has no new object for update: %v", e)
		return false
	}

	newPromo, ok := e.ObjectNew.(*argocd.Application)
	if !ok {
		p.logger.Errorf("Failed to convert new Application: %v", e.ObjectNew)
		return false
	}
	oldPromo, ok := e.ObjectOld.(*argocd.Application)
	if !ok {
		p.logger.Errorf("Failed to convert old Application: %v", e.ObjectOld)
		return false
	}

	if newPromo.Status.OperationState == nil {
		// No operation state to compare against.
		return false
	}

	newOperationCompleted := newPromo.Status.OperationState.Phase.Completed()
	oldOperationCompleted := oldPromo.Status.OperationState != nil && oldPromo.Status.OperationState.Phase.Completed()

	return newOperationCompleted && !oldOperationCompleted
}

func (p ArgoCDAppOperationCompleted) Delete(event.DeleteEvent) bool {
	return false
}

func (p ArgoCDAppOperationCompleted) Generic(event.GenericEvent) bool {
	return false
}
