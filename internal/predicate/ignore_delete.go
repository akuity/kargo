package predicate

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// IgnoreDelete is a predicate that filters out Delete events.
//
// Typically, a reconciler will not need to do anything when it receives an
// event.TypedDeleteEvent, as it acts on the event.TypedUpdateEvent which sets
// the deletion timestamp.
type IgnoreDelete[T any] struct {
	predicate.TypedFuncs[T]
}

// Delete always returns false, ignoring the event.
func (i IgnoreDelete[T]) Delete(event.TypedDeleteEvent[T]) bool {
	return false
}
