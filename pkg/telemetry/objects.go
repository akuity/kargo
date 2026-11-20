package telemetry

import (
	"reflect"
	"slices"

	"go.opentelemetry.io/otel/attribute"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// ObjectAttributes returns the attributes that identify obj: its kind and
// name, followed by KargoAttributes(obj).
func ObjectAttributes(obj metav1.Object) []attribute.KeyValue {
	if obj == nil || reflect.ValueOf(obj).IsNil() {
		return nil
	}
	attrs := []attribute.KeyValue{KindKey.String(reflect.TypeOf(obj).Elem().Name())}
	// A Secret's name is left out: it can say whose credentials it holds.
	if _, secret := obj.(*corev1.Secret); !secret && obj.GetName() != "" {
		attrs = append(attrs, NameKey.String(obj.GetName()))
	}
	return append(attrs, KargoAttributes(obj)...)
}

// kargoPkgPath is the package path of Kargo's API types.
var kargoPkgPath = reflect.TypeFor[kargoapi.Freight]().PkgPath()

// KargoAttributes returns obj's kargo.* attributes: its Project and the Kargo
// objects it names. A non-Project namespace is k8s.namespace.name.
func KargoAttributes(obj metav1.Object) []attribute.KeyValue {
	if obj == nil || reflect.ValueOf(obj).IsNil() {
		return nil
	}
	t := reflect.TypeOf(obj).Elem()
	if t.PkgPath() != kargoPkgPath {
		if obj.GetNamespace() == "" {
			return nil
		}
		return []attribute.KeyValue{NamespaceKey.String(obj.GetNamespace())}
	}
	var attrs []attribute.KeyValue
	if obj.GetNamespace() != "" {
		attrs = append(attrs, ProjectKey.String(obj.GetNamespace()))
	}
	name := obj.GetName()
	switch o := obj.(type) {
	case *kargoapi.Freight:
		attrs = append(attrs, FreightKey.String(name))
		attrs = append(attrs, FreightDetailAttributes(o)...)
	case *kargoapi.Warehouse:
		attrs = append(attrs, WarehouseKey.String(name))
	case *kargoapi.Stage:
		attrs = append(attrs, StageKey.String(name))
	case *kargoapi.Promotion:
		attrs = append(attrs,
			PromotionKey.String(name), StageKey.String(o.Spec.Stage), FreightKey.String(o.Spec.Freight),
		)
	case *kargoapi.PromotionRequest:
		attrs = append(attrs,
			PromotionRequestKey.String(name), StageKey.String(o.Spec.Stage), FreightKey.String(o.Spec.Freight),
		)
	case *kargoapi.Project:
		attrs = append(attrs, ProjectKey.String(name))
	}
	// Unset fields are left out.
	return slices.DeleteFunc(attrs, func(a attribute.KeyValue) bool {
		return a.Value.Type() == attribute.STRING && a.Value.AsString() == ""
	})
}

// FreightDetailAttributes returns a Freight's origin, its Warehouse if that is
// the origin, and its commit ID if it has exactly one.
func FreightDetailAttributes(f *kargoapi.Freight) []attribute.KeyValue {
	var attrs []attribute.KeyValue
	if f.Origin.Name != "" {
		attrs = append(attrs, FreightOriginKey.String(string(f.Origin.Kind)+"/"+f.Origin.Name))
		if f.Origin.Kind == kargoapi.FreightOriginKindWarehouse {
			attrs = append(attrs, WarehouseKey.String(f.Origin.Name))
		}
	}
	if len(f.Commits) == 1 {
		attrs = append(attrs, VCSCommitKey.String(f.Commits[0].ID))
	}
	return attrs
}
