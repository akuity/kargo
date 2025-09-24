package stages

import (
	"sigs.k8s.io/controller-runtime/pkg/event"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// IsControlFlowStage is a predicate that filters out Stages based on whether
// they are control flow Stages or not.
type IsControlFlowStage bool

func (s IsControlFlowStage) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}

	newObj, ok := e.Object.(*kargoapi.Stage)
	if !ok {
		return false
	}

	return newObj.IsControlFlow() == bool(s)
}

func (s IsControlFlowStage) Update(e event.UpdateEvent) bool {
	if e.ObjectNew == nil {
		return false
	}

	newObj, ok := e.ObjectNew.(*kargoapi.Stage)
	if !ok {
		return false
	}

	return newObj.IsControlFlow() == bool(s)
}

func (s IsControlFlowStage) Delete(event.DeleteEvent) bool {
	return false
}

func (s IsControlFlowStage) Generic(event.GenericEvent) bool {
	return false
}
