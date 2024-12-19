package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:resource:shortName={promotask,promotasks}
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

type PromotionTask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec describes the composition of a PromotionTask, including the
	// variables available to the task and the steps.
	//
	// +kubebuilder:validation:Required
	Spec PromotionTaskSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

type PromotionTaskSpec struct {
	// Vars specifies the variables available to the PromotionTask. The
	// values of these variables are the default values that can be
	// overridden by the step referencing the task.
	Vars []PromotionVariable `json:"vars,omitempty" protobuf:"bytes,1,rep,name=vars"`
	// Steps specifies the directives to be executed as part of this
	// PromotionTask. The steps as defined here are inflated into a
	// Promotion when it is built from a PromotionTemplate.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:items:XValidation:message="PromotionTask step must have uses set and must not reference another task",rule="has(self.uses) && !has(self.task)"
	Steps []PromotionStep `json:"steps" protobuf:"bytes,2,rep,name=steps"`
}

// +kubebuilder:object:root=true

// PromotionTaskList contains a list of PromotionTasks.
type PromotionTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []PromotionTask `json:"items" protobuf:"bytes,2,rep,name=items"`
}
