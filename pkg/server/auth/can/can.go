// Package can describes the operations the API server asks permission for,
// so that a permission check reads as the question it asks:
//
//	s.authorize(ctx, can.Promote().Stage(project, stage))
//
// Building an Access checks nothing. It is a plain value, which the
// authorizing client answers for the identity bound to the request context.
package can

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Verb is what an Access asks to do.
type Verb string

// Do returns the named verb, for checks whose verb is only known at runtime.
func Do(verb string) Verb { return Verb(verb) }

// Get returns the "get" verb.
func Get() Verb { return "get" }

// List returns the "list" verb.
func List() Verb { return "list" }

// Watch returns the "watch" verb.
func Watch() Verb { return "watch" }

// Create returns the "create" verb.
func Create() Verb { return "create" }

// Update returns the "update" verb.
func Update() Verb { return "update" }

// Patch returns the "patch" verb.
func Patch() Verb { return "patch" }

// Delete returns the "delete" verb.
func Delete() Verb { return "delete" }

// DeleteCollection returns the "deletecollection" verb.
func DeleteCollection() Verb { return "deletecollection" }

// Promote returns Kargo's custom "promote" verb, which Kubernetes does not
// check implicitly, so handlers that promote must ask for it explicitly.
func Promote() Verb { return "promote" }

// Access is an operation to authorize: a verb on a resource.
type Access struct {
	Verb        string
	Resource    schema.GroupVersionResource
	Subresource string
	Key         types.NamespacedName
}

// Resource returns the Verb on any resource, for checks whose resource is only
// known at runtime. The key's namespace is empty for a cluster-scoped resource
// and its name is empty for an operation on a collection.
func (v Verb) Resource(
	gvr schema.GroupVersionResource,
	key types.NamespacedName,
) Access {
	return Access{Verb: string(v), Resource: gvr, Key: key}
}

// Kargo returns the Verb on a collection of Kargo resources, named by their
// lowercase plural, e.g. "stages". In narrows it to one namespace.
func (v Verb) Kargo(resource string) Access {
	return v.kargo(resource, "", "")
}

// Stage returns the Verb on the named Stage in the Project.
func (v Verb) Stage(project, name string) Access {
	return v.kargo("stages", project, name)
}

// Promotion returns the Verb on the named Promotion in the Project.
func (v Verb) Promotion(project, name string) Access {
	return v.kargo("promotions", project, name)
}

// PromotionRequest returns the Verb on the named PromotionRequest in the
// Project.
func (v Verb) PromotionRequest(project, name string) Access {
	return v.kargo("promotionrequests", project, name)
}

// Warehouse returns the Verb on the named Warehouse in the Project.
func (v Verb) Warehouse(project, name string) Access {
	return v.kargo("warehouses", project, name)
}

// Target returns the Verb on the named Target in the Project. Targets live in
// the database rather than in Kubernetes, but RBAC rules name them like any
// other Kargo resource. An empty name addresses the Project's collection.
func (v Verb) Target(project, name string) Access {
	return v.kargo("targets", project, name)
}

// ProjectConfig returns the Verb on the Project's ProjectConfig, which is
// named after the Project.
func (v Verb) ProjectConfig(project string) Access {
	return v.kargo("projectconfigs", project, project)
}

// ClusterConfig returns the Verb on the named, cluster-scoped ClusterConfig.
func (v Verb) ClusterConfig(name string) Access {
	return v.kargo("clusterconfigs", "", name)
}

func (v Verb) kargo(resource, namespace, name string) Access {
	return v.Resource(
		kargoapi.GroupVersion.WithResource(resource),
		types.NamespacedName{Namespace: namespace, Name: name},
	)
}

// In returns the Access narrowed to the namespace.
func (a Access) In(namespace string) Access {
	a.Key.Namespace = namespace
	return a
}
