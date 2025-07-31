package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// ResponsibleFor is an implementation of predicate.Predicate used by
// reconcilers to narrow the set of client.Objects they watch according to how
// they are labeled.
type ResponsibleFor[T any] struct {
	IsDefaultController bool
	ShardName           string
}

// Create implements predicate.Predicate.
func (i ResponsibleFor[T]) Create(e event.TypedCreateEvent[T]) bool {
	obj := any(e.Object).(client.Object) // nolint: forcetypeassert
	if obj == nil {
		return false
	}
	return i.IsResponsible(obj)
}

// Update implements predicate.Predicate.
func (i ResponsibleFor[T]) Update(e event.TypedUpdateEvent[T]) bool {
	obj := any(e.ObjectNew).(client.Object) // nolint: forcetypeassert
	if obj == nil {
		return false
	}
	return i.IsResponsible(obj)
}

// Delete implements predicate.Predicate.
func (i ResponsibleFor[T]) Delete(e event.TypedDeleteEvent[T]) bool {
	obj := any(e.Object).(client.Object) // nolint: forcetypeassert
	if obj == nil {
		return false
	}
	return i.IsResponsible(obj)
}

// Generic implements predicate.Predicate.
func (i ResponsibleFor[T]) Generic(e event.TypedGenericEvent[T]) bool {
	obj := any(e.Object).(client.Object) // nolint: forcetypeassert
	if obj == nil {
		return false
	}
	return i.IsResponsible(obj)
}

func (i ResponsibleFor[T]) IsResponsible(obj client.Object) bool {
	objShard := obj.GetLabels()[kargoapi.LabelKeyShard]
	return objShard == i.ShardName || (objShard == "" && i.IsDefaultController)
}
