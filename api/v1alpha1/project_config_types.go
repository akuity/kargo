package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

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
	PromotionPolicies []PromotionPolicy `json:"promotionPolicies,omitempty" protobuf:"bytes,1,rep,name=promotionPolicies"`
	// Receivers defines the receivers that are used to receive warehouse events
	// and trigger refreshes.
	Receivers []Receiver `json:"receivers,omitempty" protobuf:"bytes,2,rep,name=receivers"`
}

// PromotionPolicy defines policies governing the promotion of Freight to a
// specific Stage.
//
// +kubebuilder:validation:XValidation:message="PromotionPolicy must have exactly one of stage or stageSelector set",rule="has(self.stage) ? !has(self.stageSelector) : has(self.stageSelector)"
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

type Receiver struct {
	// Name is the name of the receiver.
	//
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	// Type is the type of the receiver.
	//
	// TODO: Add more receiver enum types(e.g. Dockerhub, Quay, Gitlab, etc...)
	// +kubebuilder:validation:Enum=GitHub;
	Type string `json:"type,omitempty" protobuf:"bytes,2,opt,name=type"`
	// URL is the URL of the receiver.
	//
	// +kubebuilder:validation:Format=uri
	URL string `json:"url,omitempty" protobuf:"bytes,3,opt,name=url"`
	// Secret is the name of the secret that contains the credentials for the
	// receiver.
	//
	// +kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	SecretRef string `json:"secretRef,omitempty" protobuf:"bytes,4,opt,name=secretRef"`
}

// PromotionPolicySelector is a selector that matches the resource to which
// this policy applies. It can be used to match a specific resource by name or
// to match a set of resources by label.
type PromotionPolicySelector struct {
	// Name is the name of the resource to which this policy applies.
	//
	// It can be an exact name, a regex pattern (with prefix "regex:"), or a
	// glob pattern (with prefix "glob:").
	//
	// When both Name and LabelSelector are specified, the Name is ANDed with
	// the LabelSelector. I.e., the resource must match both the Name and
	// LabelSelector to be selected by this policy.
	//
	// NOTE: Using a specific exact name is the most secure option. Pattern
	// matching via regex or glob can be exploited by users with permissions to
	// match promotion policies that weren't intended to apply to their
	// resources. For example, a user could create a resource with a name
	// deliberately crafted to match the pattern, potentially bypassing intended
	// promotion controls.
	//
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`

	// LabelSelector is a selector that matches the resource to which this policy
	// applies.
	//
	// When both Name and LabelSelector are specified, the Name is ANDed with
	// the LabelSelector. I.e., the resource must match both the Name and
	// LabelSelector to be selected by this policy.
	//
	// NOTE: Using label selectors introduces security risks as users with
	// appropriate permissions could create new resources with labels that match
	// the selector, potentially enabling unauthorized auto-promotion.
	// For sensitive environments, exact Name matching provides tighter control.
	*metav1.LabelSelector `json:",inline" protobuf:"bytes,2,opt,name=labelSelector"`
}

// +kubebuilder:object:root=true

// ProjectConfigList is a list of ProjectConfig resources.
type ProjectConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []ProjectConfig `json:"items" protobuf:"bytes,2,rep,name=items"`
}
