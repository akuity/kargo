package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PromotionPhase string

const (
	PromotionPhasePending    PromotionPhase = "Pending"
	PromotionPhaseInProgress PromotionPhase = "Promoting"
	PromotionPhaseComplete   PromotionPhase = "Completed"
	PromotionPhaseFailed     PromotionPhase = "Failed"
)

//+kubebuilder:resource:shortName={promo,promos}
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name=Stage,type=string,JSONPath=`.spec.stage`
//+kubebuilder:printcolumn:name=State,type=string,JSONPath=`.spec.state`
//+kubebuilder:printcolumn:name=Phase,type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// Promotion represents a request to transition a particular Stage into a
// particular state.
type Promotion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec describes the desired transition of a specific Stage into a specific
	// StageState.
	//
	//+kubebuilder:validation:Required
	Spec *PromotionSpec `json:"spec"`
	// Status describes the current state of the transition represented by this
	// Promotion.
	Status PromotionStatus `json:"status,omitempty"`
}

// PromotionSpec describes the desired transition of a specific Stage into a
// specific StageState.
type PromotionSpec struct {
	// Stage specifies the name of the Stage to which this Promotion
	// applies. The Stage referenced by this field MUST be in the same
	// namespace as the Promotion.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Stage string `json:"stage"`
	// State specifies the specific StageState into which the Stage referenced by
	// the Stage field should be transitioned. The State MUST be among the Stage's
	// Status.AvailableStates or the Promotion will ultimately fail.
	//
	//
	//+kubebuilder:validation:MinLength=1
	State string `json:"state"`
}

// PromotionStatus describes the current state of the transition represented by
// a Promotion.
type PromotionStatus struct {
	// Phase describes where the Promotion currently is in its lifecycle.
	Phase PromotionPhase `json:"phase,omitempty"`
	// Error describes any errors that are preventing the Promotion controller
	// from executing this Promotion. i.e. If the Phase field has a value of
	// Failed, this field can be expected to explain why.
	Error string `json:"error,omitempty"`
}

//+kubebuilder:object:root=true

// PromotionList contains a list of Promotion
type PromotionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Promotion `json:"items"`
}
