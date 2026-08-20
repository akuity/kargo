package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// Target represents a single destination -- a cluster, for instance -- that
// Stages promote Freight to. A Target is purely descriptive: it holds
// target-specific values consumed by the promotion steps of Stages that
// govern it and records which Stages those are. It defines no promotion
// steps and no Freight sources of its own and therefore cannot effect any
// promotion itself.
type Target struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec describes the Target.
	Spec TargetSpec `json:"spec,omitempty"`
	// Status describes the current status of the Target.
	Status TargetStatus `json:"status,omitempty"`
}

func (t *Target) GetStatus() *TargetStatus {
	return &t.Status
}

// TargetSpec describes a Target.
type TargetSpec struct {
	// Params is a map of arbitrary, target-specific values. Values may be any
	// valid JSON -- including nested objects and arrays -- so promotion steps
	// can reference deeply nested data. Promotion steps of Stages that govern
	// this Target may reference these values by key in their expressions (for
	// example, target.params.branch or target.params.cluster.region).
	//
	// +optional
	Params map[string]apiextensionsv1.JSON `json:"params,omitempty"`
}

// TargetStatus describes the current status of a Target. It is intentionally
// empty for now; fields will be added as they are needed.
type TargetStatus struct{}

// +kubebuilder:object:root=true

// TargetList is a list of Target resources.
type TargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Target `json:"items"`
}
