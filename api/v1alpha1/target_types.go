package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].message"
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// Target represents a single destination -- a cluster, for instance -- that
// Stages promote Freight to. A Target is purely descriptive: it holds
// target-specific values consumed by the promotion steps of Stages that
// govern it and records which Stages those are. It defines no promotion
// steps and no Freight sources of its own and therefore cannot effect any
// promotion itself.
type Target struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Spec describes the Target.
	Spec TargetSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
	// Status describes the current status of the Target.
	Status TargetStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
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
	Params map[string]apiextensionsv1.JSON `json:"params,omitempty" protobuf:"bytes,1,rep,name=params" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

// TargetStatus describes the current status of a Target.
type TargetStatus struct {
	// Conditions contains the last observations of the Target's current state.
	//
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchMergeKey:"type" patchStrategy:"merge" protobuf:"bytes,1,rep,name=conditions"`
	// OwnedBy describes the Stages that currently govern this Target, with one
	// entry per (Stage, Freight origin) pair.
	OwnedBy []TargetOwnership `json:"ownedBy,omitempty" protobuf:"bytes,2,rep,name=ownedBy"`
}

// GetConditions implements the conditions.Getter interface.
func (t *TargetStatus) GetConditions() []metav1.Condition {
	return t.Conditions
}

// SetConditions implements the conditions.Setter interface.
func (t *TargetStatus) SetConditions(conditions []metav1.Condition) {
	t.Conditions = conditions
}

// TargetOwnership records one Stage's governance of a Target with respect to
// Freight from a single origin.
type TargetOwnership struct {
	// Stage is the name of the governing Stage.
	Stage string `json:"stage,omitempty" protobuf:"bytes,1,opt,name=stage"`
	// Origin is the origin of the Freight that the governing Stage promotes to
	// this Target.
	Origin FreightOrigin `json:"origin,omitempty" protobuf:"bytes,2,opt,name=origin"`
	// CurrentFreight is the name of the Freight from Origin most recently
	// promoted to this Target by the governing Stage.
	CurrentFreight string `json:"currentFreight,omitempty" protobuf:"bytes,3,opt,name=currentFreight"`
}

// +kubebuilder:object:root=true

// TargetList is a list of Target resources.
type TargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Target `json:"items" protobuf:"bytes,2,rep,name=items"`
}
