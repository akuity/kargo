package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:resource:shortName={targetpromo,targetpromos}
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=Shard,type=string,JSONPath=`.metadata.labels.kargo\.akuity\.io/shard`
// +kubebuilder:printcolumn:name=Stage,type=string,JSONPath=`.spec.stage`
// +kubebuilder:printcolumn:name=Target,type=string,JSONPath=`.spec.target`
// +kubebuilder:printcolumn:name=Freight,type=string,JSONPath=`.spec.freight`
// +kubebuilder:printcolumn:name=Phase,type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// TargetPromotion represents the execution of a Promotion's pipeline against a
// single Target of the Stage being promoted to. A Promotion creates one
// TargetPromotion per Target it runs against, owning each so they are
// garbage-collected with the parent Promotion.
type TargetPromotion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Spec describes the desired transition of a specific Stage into a specific
	// Freight against a specific Target.
	//
	// +kubebuilder:validation:Required
	Spec TargetPromotionSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	// Status describes the current state of the transition represented by this
	// TargetPromotion.
	Status PromotionStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

func (p *TargetPromotion) GetStatus() *PromotionStatus {
	return &p.Status
}

// TargetPromotionSpec describes the desired transition of a specific Stage into
// a specific Freight against a specific Target. It mirrors PromotionSpec, adding
// the Target the pipeline runs against.
type TargetPromotionSpec struct {
	// Stage specifies the name of the Stage to which this TargetPromotion
	// applies. The Stage referenced by this field MUST be in the same namespace
	// as the TargetPromotion.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +akuity:test-kubebuilder-pattern=KubernetesName
	Stage string `json:"stage" protobuf:"bytes,1,opt,name=stage"`
	// Freight specifies the piece of Freight to be promoted into the Stage
	// referenced by the Stage field.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +akuity:test-kubebuilder-pattern=KubernetesName
	Freight string `json:"freight" protobuf:"bytes,2,opt,name=freight"`
	// Target specifies the name of the Target that this TargetPromotion's
	// pipeline runs against. The Target referenced by this field MUST be in the
	// same namespace as the TargetPromotion.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +akuity:test-kubebuilder-pattern=KubernetesName
	Target string `json:"target" protobuf:"bytes,3,opt,name=target"`
	// Vars is a list of variables that can be referenced by expressions in
	// promotion steps.
	Vars []ExpressionVariable `json:"vars,omitempty" protobuf:"bytes,4,rep,name=vars"`
	// Steps specifies the directives to be executed as part of this
	// TargetPromotion. The order in which the directives are executed is the
	// order in which they are listed in this field.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:items:XValidation:message="Promotion step must have uses set and must not reference a task",rule="has(self.uses) && !has(self.task)"
	Steps []PromotionStep `json:"steps" protobuf:"bytes,5,rep,name=steps"`
}

// +kubebuilder:object:root=true

// TargetPromotionList contains a list of TargetPromotion resources.
type TargetPromotionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []TargetPromotion `json:"items" protobuf:"bytes,2,rep,name=items"`
}
