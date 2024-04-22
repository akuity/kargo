package rbac

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

// Role is a wrapper around a svcv1alpha1.Role to implement runtime.Object. This
// can be used to adapt a svcv1alpha1.Role for use with printers from
// cli-runtime.
type Role struct {
	metav1.ObjectMeta
	*svcv1alpha1.Role
}

// NewRole wraps a svcv1alpha1.Role in a Role to adapt it for use with printers
// from cli-runtime.
func NewRole(role *svcv1alpha1.Role) *Role {
	return &Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: role.Project,
			Name:      role.Name,
			CreationTimestamp: metav1.Time{
				Time: role.CreationTimestamp.AsTime(),
			},
		},
		Role: role,
	}
}

// DeepCopyObject implements runtime.Object.
func (r *Role) DeepCopyObject() runtime.Object {
	return &Role{
		Role: r.Role,
	}
}

// GetObjectKind implements runtime.Object.
func (r *Role) GetObjectKind() schema.ObjectKind {
	return roleKind{}
}

// roleKind is an implementation of the schema.ObjectKind interface for Role.
type roleKind struct{}

// SetGroupVersionKind implements schema.ObjectKind.
func (r roleKind) SetGroupVersionKind(schema.GroupVersionKind) {}

// GroupVersionKind implements schema.ObjectKind.
func (r roleKind) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "rbac.kargo.akuity.io",
		Version: "v1alpha1",
		Kind:    "Role",
	}
}

// RoleResources is a wrapper around a svcv1alpha1.RoleResources to implement
// runtime.Object. This can be used to adapt a svcv1alpha1.RoleResources for use
// with printers from cli-runtime.
type RoleResources struct {
	*svcv1alpha1.RoleResources
}

// DeepCopyObject implements runtime.Object.
func (r *RoleResources) DeepCopyObject() runtime.Object {
	return &RoleResources{
		RoleResources: r.RoleResources,
	}
}

// GetObjectKind implements runtime.Object.
func (r *RoleResources) GetObjectKind() schema.ObjectKind {
	return roleResourcesKind{}
}

// roleResourcesKind is an implementation of the schema.ObjectKind interface for
// RoleResources.
type roleResourcesKind struct{}

// SetGroupVersionKind implements schema.ObjectKind.
func (r roleResourcesKind) SetGroupVersionKind(schema.GroupVersionKind) {}

// GroupVersionKind implements schema.ObjectKind.
func (r roleResourcesKind) GroupVersionKind() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "rbac.kargo.akuity.io",
		Version: "v1alpha1",
		Kind:    "RoleResources",
	}
}
