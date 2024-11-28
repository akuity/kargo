package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// PromotionTemplate defines a template for a Promotion that can be used to
// incorporate Freight into a Stage.
type PromotionTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Spec describes how to create a Promotion for a Stage using this template.
	//
	// +kubebuilder:validation:Required
	Spec PromotionTemplateSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

// PromotionTemplateSpec describes the (partial) specification of a Promotion
// for a Stage. This is a template that can be used to create a Promotion for a
// Stage.
type PromotionTemplateSpec struct {
	// Vars is a list of variables that can be referenced by expressions in
	// promotion steps.
	Vars []PromotionVariable `json:"vars,omitempty" protobuf:"bytes,2,rep,name=vars"`
	// Steps specifies the directives to be executed as part of a Promotion.
	// The order in which the directives are executed is the order in which they
	// are listed in this field.
	//
	// +kubebuilder:validation:MinItems=1
	Steps []PromotionStep `json:"steps,omitempty" protobuf:"bytes,1,rep,name=steps"`
}
