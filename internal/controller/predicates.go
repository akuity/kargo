package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

type ResponsibleFor[T any] struct {
	IsDefaultController bool
	ShardName           string
}

func (i ResponsibleFor[T]) Create(e event.TypedCreateEvent[T]) bool {
	obj := any(e.Object).(client.Object) // nolint: forcetypeassert
	if obj == nil {
		return false
	}
	return i.isMyResponsibility(obj)
}

func (i ResponsibleFor[T]) Update(e event.TypedUpdateEvent[T]) bool {
	obj := any(e.ObjectNew).(client.Object) // nolint: forcetypeassert
	if obj == nil {
		return false
	}
	return i.isMyResponsibility(obj)
}

func (i ResponsibleFor[T]) Delete(e event.TypedDeleteEvent[T]) bool {
	obj := any(e.Object).(client.Object) // nolint: forcetypeassert
	if obj == nil {
		return false
	}
	return i.isMyResponsibility(obj)
}

func (i ResponsibleFor[T]) Generic(e event.TypedGenericEvent[T]) bool {
	obj := any(e.Object).(client.Object) // nolint: forcetypeassert
	if obj == nil {
		return false
	}
	return i.isMyResponsibility(obj)
}

func (i ResponsibleFor[T]) isMyResponsibility(obj client.Object) bool {
	objShard := obj.GetLabels()[kargoapi.LabelKeyShard]
	return objShard == i.ShardName || objShard == "" && i.IsDefaultController
}
