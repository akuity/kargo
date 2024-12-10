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

	// Spec describes the composition of a PromotionTask, including the inputs
	// available to the task and the steps.
	//
	// +kubebuilder:validation:Required
	Spec PromotionTaskSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
}

type PromotionTaskSpec struct {
	// Inputs specifies the inputs available to the PromotionTask. These inputs
	// can be specified in the PromotionTemplate as configuration for the task,
	// and can be used in the Steps to parameterize the execution of the task.
	Inputs []PromotionTaskInput `json:"inputs,omitempty" protobuf:"bytes,1,rep,name=inputs"`
	// Steps specifies the directives to be executed as part of this
	// PromotionTask. The steps as defined here are deflated into a
	// Promotion when it is built from a PromotionTemplate.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Steps []PromotionStep `json:"steps,omitempty" protobuf:"bytes,2,rep,name=steps"`
}

// PromotionTaskInput defines an input parameter for a PromotionTask. This input
// can be specified in the PromotionTemplate as configuration for the task, and
// can be used in the Steps to parameterize the execution of the task.
type PromotionTaskInput struct {
	// Name of the configuration parameter, which should be unique within the
	// PromotionTask. This name can be used to reference the parameter in the
	// PromotionTaskSpec.Steps.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Default specifies a default value for the parameter. This value will be
	// used if the parameter is not specified in the PromotionTemplate.
	// If left unspecified, the input value is required to be specified in the
	// configuration of the step referencing this task.
	//
	// +kubebuilder:validation:Optional
	Default string `json:"default,omitempty" protobuf:"bytes,2,opt,name=default"`
}

// +kubebuilder:object:root=true

// PromotionTaskList contains a list of PromotionTasks.
type PromotionTaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []PromotionTaskList `json:"items" protobuf:"bytes,2,rep,name=items"`
}
