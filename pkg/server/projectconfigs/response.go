package projectconfigs

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// Resource is a ProjectConfig as the API serves it: with apiVersion and kind.
// kargoapi.ProjectConfig embeds no TypeMeta, so that controller-gen emits no
// CRD for it, but clients need both, for instance to send what they read
// back through the generic resources endpoints, which route a manifest by
// its kind. Resource is also a runtime.Object, so that clients can print it
// as they print Kubernetes resources.
type Resource struct {
	metav1.TypeMeta        `json:",inline"`
	kargoapi.ProjectConfig `json:",inline"`
} // @name ProjectConfig

// ResponseBody returns cfg as it is served to clients, with apiVersion and
// kind set. Every response that carries a ProjectConfig serializes this.
func ResponseBody(cfg *kargoapi.ProjectConfig) *Resource {
	r := &Resource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       kind,
		},
	}
	if cfg != nil {
		r.ProjectConfig = *cfg
	}
	return r
}

// DeepCopyObject implements runtime.Object.
func (r *Resource) DeepCopyObject() runtime.Object {
	if r == nil {
		return nil
	}
	return &Resource{
		TypeMeta:      r.TypeMeta,
		ProjectConfig: *r.DeepCopy(),
	}
}
