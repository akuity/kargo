package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// Target is a resource type that describes a single destination -- such as a
// cluster -- to which Freight may be promoted. Targets are typically selected
// by label, with each selected Target's parameters used to shape an otherwise
// identical promotion process for that specific destination.
type Target struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Spec describes the Target.
	Spec TargetSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// TargetSpec describes a Target.
type TargetSpec struct {
	// Params is a map of arbitrary, user-defined parameters describing the
	// Target -- for example, a cluster's address or the name of a branch
	// containing that cluster's configuration. Keys are strings; values may be
	// any valid JSON, including deeply nested objects and arrays.
	//
	// +optional
	Params map[string]apiextensionsv1.JSON `json:"params,omitempty" protobuf:"bytes,1,rep,name=params" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"` // nolint: lll
}

// +kubebuilder:object:root=true

// TargetList is a list of Target resources.
type TargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Target `json:"items" protobuf:"bytes,2,rep,name=items"`
}
