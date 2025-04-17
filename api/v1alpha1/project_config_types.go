package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:subresource:status

// ProjectConfig is a resource type that describes the configuration of a
// Project.
type ProjectConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Spec describes the configuration of a Project.
	Spec ProjectConfigSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// ProjectConfigSpec describes the configuration of a Project.
type ProjectConfigSpec struct {
	// PromotionPolicies defines policies governing the promotion of Freight to
	// specific Stages within the Project.
	//
	// +kubebuilder:validation:XValidation:message="PromotionPolicy must have exactly one of stage or stageSelector set",rule="has(self.stage) ? !has(self.stageSelector) : has(self.stageSelector)"
	PromotionPolicies []PromotionPolicy `json:"promotionPolicies,omitempty" protobuf:"bytes,1,rep,name=promotionPolicies"`
}

// PromotionPolicy defines policies governing the promotion of Freight to a
// specific Stage.
type PromotionPolicy struct {
	// Stage is the name of the Stage to which this policy applies.
	//
	// Deprecated: Use StageSelector instead.
	//
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Stage string `json:"stage,omitempty" protobuf:"bytes,1,opt,name=stage"`
	// StageSelector is a selector that matches the Stage resource to which
	// this policy applies.
	StageSelector *PromotionPolicySelector `json:"stageSelector,omitempty" protobuf:"bytes,3,opt,name=stageSelector"`
	// AutoPromotionEnabled indicates whether new Freight can automatically be
	// promoted into the Stage referenced by the Stage field. Note: There are may
	// be other conditions also required for an auto-promotion to occur. This
	// field defaults to false, but is commonly set to true for Stages that
	// subscribe to Warehouses instead of other, upstream Stages. This allows
	// users to define Stages that are automatically updated as soon as new
	// artifacts are detected.
	AutoPromotionEnabled bool `json:"autoPromotionEnabled,omitempty" protobuf:"varint,2,opt,name=autoPromotionEnabled"`
}

type PromotionPolicySelector struct {
	// Name is the name of the resource to which this policy applies.
	//
	// It selects a resource based on its name. It supports exact name matching,
	// regex matching (with prefix "regex:"), or glob pattern matching (with
	// prefix "glob:").
	//
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// LabelSelector is a selector that matches the resource to which this policy
	// applies.
	*metav1.LabelSelector `json:",inline" protobuf:"bytes,2,opt,name=labelSelector"`
}

// +kubebuilder:object:root=true

// ProjectConfigList is a list of ProjectConfig resources.
type ProjectConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []ProjectConfig `json:"items" protobuf:"bytes,2,rep,name=items"`
}
