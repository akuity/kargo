package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PromotionSet groups the Promotions that fan out a Stage's selected Freight
// to its selected Targets.
//
// +kubebuilder:resource:scope=Namespaced,shortName={promotionset,promotionsets}
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Targets",type="integer",JSONPath=".status.targetCount"
// +kubebuilder:printcolumn:name="Succeeded",type="integer",JSONPath=".status.succeededTargetCount"
// +kubebuilder:printcolumn:name="Failed",type="integer",JSONPath=".status.failedTargetCount"
// +kubebuilder:printcolumn:name="Errored",type="integer",JSONPath=".status.erroredTargetCount"
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:subresource:status
type PromotionSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec describes the Stage and Freight associated with the PromotionSet.
	//
	// +kubebuilder:validation:Required
	Spec PromotionSetSpec `json:"spec"`

	// Status describes the current aggregate state of the PromotionSet's
	// Promotions.
	//
	// +kubebuilder:validation:Optional
	Status PromotionSetStatus `json:"status,omitempty"`
}

// GetStatus returns the PromotionSet's status.
func (p *PromotionSet) GetStatus() *PromotionSetStatus {
	return &p.Status
}

// GetConditions returns the PromotionSet's conditions.
func (p *PromotionSet) GetConditions() []metav1.Condition {
	return p.Status.Conditions
}

// SetConditions sets the PromotionSet's conditions.
func (p *PromotionSet) SetConditions(conditions []metav1.Condition) {
	p.Status.Conditions = conditions
}

// PromotionSetSpec describes the Stage and Freight associated with a
// PromotionSet.
type PromotionSetSpec struct {
	// Stage specifies the name of the Stage that promotes the Freight.
	// The Stage MUST be in the same namespace as the PromotionSet.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +akuity:test-kubebuilder-pattern=KubernetesName
	Stage string `json:"stage"`

	// Freight specifies the piece of Freight promoted by the Stage.
	// The Freight MUST be in the same namespace as the PromotionSet.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +akuity:test-kubebuilder-pattern=KubernetesName
	Freight string `json:"freight"`

	// Targets specifies the Targets to which this PromotionSet promotes Freight.
	// Each Target MUST be in the same namespace as the PromotionSet.
	//
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	Targets []PromotionSetTarget `json:"targets"`

	// Suspended prevents this PromotionSet from creating Promotions for newly
	// specified Targets. Existing Promotions are not affected.
	//
	// +kubebuilder:validation:Optional
	Suspended bool `json:"suspended,omitempty"`
}

// PromotionSetTarget identifies a Target selected by a PromotionSet.
type PromotionSetTarget struct {
	// Name is the name of the Target.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +akuity:test-kubebuilder-pattern=KubernetesName
	Name string `json:"name"`
}

// PromotionSetStatus describes the observed aggregate state of a
// PromotionSet's Promotions.
type PromotionSetStatus struct {
	// ObservedGeneration is the generation of the spec last reconciled.
	//
	// +kubebuilder:validation:Optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions contains the last observations of the PromotionSet's current
	// state.
	//
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchMergeKey:"type" patchStrategy:"merge"`

	// TargetCount is the number of Targets selected by the PromotionSet.
	//
	// +kubebuilder:validation:Optional
	TargetCount int32 `json:"targetCount,omitempty"`

	// PendingTargetCount is the number of Targets whose latest Promotion
	// attempt is in the Pending phase.
	//
	// +kubebuilder:validation:Optional
	PendingTargetCount int32 `json:"pendingTargetCount,omitempty"`

	// RunningTargetCount is the number of Targets whose latest Promotion
	// attempt is in the Running phase.
	//
	// +kubebuilder:validation:Optional
	RunningTargetCount int32 `json:"runningTargetCount,omitempty"`

	// SucceededTargetCount is the number of Targets whose latest Promotion
	// attempt is in the Succeeded phase.
	//
	// +kubebuilder:validation:Optional
	SucceededTargetCount int32 `json:"succeededTargetCount,omitempty"`

	// FailedTargetCount is the number of Targets whose latest Promotion attempt
	// is in the Failed phase.
	//
	// +kubebuilder:validation:Optional
	FailedTargetCount int32 `json:"failedTargetCount,omitempty"`

	// ErroredTargetCount is the number of Targets whose latest Promotion attempt
	// is in the Errored phase.
	//
	// +kubebuilder:validation:Optional
	ErroredTargetCount int32 `json:"erroredTargetCount,omitempty"`

	// AbortedTargetCount is the number of Targets whose latest Promotion attempt
	// is in the Aborted phase.
	//
	// +kubebuilder:validation:Optional
	AbortedTargetCount int32 `json:"abortedTargetCount,omitempty"`

	// UnknownTargetCount is the number of Targets whose latest Promotion attempt
	// has an empty or unrecognized phase.
	//
	// +kubebuilder:validation:Optional
	UnknownTargetCount int32 `json:"unknownTargetCount,omitempty"`
}

// GetConditions returns the PromotionSet status conditions.
func (s *PromotionSetStatus) GetConditions() []metav1.Condition {
	return s.Conditions
}

// SetConditions sets the PromotionSet status conditions.
func (s *PromotionSetStatus) SetConditions(conditions []metav1.Condition) {
	s.Conditions = conditions
}

// +kubebuilder:object:root=true

// PromotionSetList contains a list of PromotionSets.
type PromotionSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PromotionSet `json:"items"`
}
