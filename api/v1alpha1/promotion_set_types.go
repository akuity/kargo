package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PromotionSetPhasePending denotes a PromotionSet that has not started
	// creating Promotions.
	PromotionSetPhasePending PromotionSetPhase = "Pending"
	// PromotionSetPhaseRunning denotes a PromotionSet that is creating or
	// monitoring Promotions.
	PromotionSetPhaseRunning PromotionSetPhase = "Running"
	// PromotionSetPhaseSucceeded denotes a PromotionSet whose Promotions all
	// completed successfully.
	PromotionSetPhaseSucceeded PromotionSetPhase = "Succeeded"
	// PromotionSetPhaseFailed denotes a PromotionSet with Promotions that failed
	// for non-technical reasons.
	PromotionSetPhaseFailed PromotionSetPhase = "Failed"
	// PromotionSetPhaseErrored denotes a PromotionSet with Promotions that
	// encountered technical errors.
	PromotionSetPhaseErrored PromotionSetPhase = "Errored"
)

// PromotionSetPhase is a high-level summary of a PromotionSet's lifecycle.
type PromotionSetPhase string

// IsTerminal returns true if the PromotionSetPhase is a terminal one.
func (p *PromotionSetPhase) IsTerminal() bool {
	switch *p {
	case PromotionSetPhaseSucceeded,
		PromotionSetPhaseFailed,
		PromotionSetPhaseErrored:
		return true
	default:
		return false
	}
}

// PromotionSet groups the Promotions that fan out a Stage's selected Freight
// to its selected Targets.
//
// +kubebuilder:resource:scope=Namespaced,shortName={promotionset,promotionsets}
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name=Stage,type=string,JSONPath=`.spec.stage`
// +kubebuilder:printcolumn:name=Freight,type=string,JSONPath=`.spec.freight`
// +kubebuilder:printcolumn:name=Phase,type=string,JSONPath=`.status.phase`
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
	// +kubebuilder:validation:XValidation:rule="self == oldSelf"
	Stage string `json:"stage"`

	// Freight specifies the piece of Freight promoted by the Stage.
	// The Freight MUST be in the same namespace as the PromotionSet.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +akuity:test-kubebuilder-pattern=KubernetesName
	// +kubebuilder:validation:XValidation:rule="self == oldSelf"
	Freight string `json:"freight"`

	// Targets specifies the Targets to which this PromotionSet promotes Freight.
	// Each Target MUST be in the same namespace as the PromotionSet. The list
	// may be empty, which records that the governing Stage's target selectors
	// matched no Targets at the time the PromotionSet was created.
	//
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf"
	Targets []PromotionSetTarget `json:"targets"`
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
	// Conditions contains the last observations of the PromotionSet's current
	// state.
	//
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchMergeKey:"type" patchStrategy:"merge"`

	// Phase is a high-level summary of the PromotionSet's lifecycle.
	//
	// +kubebuilder:validation:Optional
	Phase PromotionSetPhase `json:"phase,omitempty"`

	// Summary aggregates the phases of this PromotionSet's child Promotions.
	//
	// +kubebuilder:validation:Optional
	Summary *PromotionSetSummary `json:"summary,omitempty"`

	// ObservedGeneration is the generation of the spec last reconciled.
	//
	// +kubebuilder:validation:Optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// StartedAt is the time at which the PromotionSet started.
	//
	// +kubebuilder:validation:Optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`

	// FinishedAt is the time at which the PromotionSet completed.
	FinishedAt *metav1.Time `json:"finishedAt,omitempty"`
}

// PromotionSetSummary aggregates the phases of a PromotionSet's child
// Promotions.
type PromotionSetSummary struct {
	// Pending is the number of child Promotions in the Pending phase.
	Pending int32 `json:"pending,omitempty"`
	// Running is the number of child Promotions in the Running phase.
	Running int32 `json:"running,omitempty"`
	// Succeeded is the number of child Promotions in the Succeeded phase.
	Succeeded int32 `json:"succeeded,omitempty"`
	// Failed is the number of child Promotions in the Failed phase.
	Failed int32 `json:"failed,omitempty"`
	// Errored is the number of child Promotions in the Errored phase.
	Errored int32 `json:"errored,omitempty"`
	// Aborted is the number of child Promotions in the Aborted phase.
	Aborted int32 `json:"aborted,omitempty"`
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
