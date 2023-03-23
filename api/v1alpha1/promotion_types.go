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

// Promotion represents a request to transition a particular Environment into a
// particular state.
type Promotion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec describes the desired transition of a specific Environment into a
	// specific EnvironmentState.
	//
	//+kubebuilder:validation:Required
	Spec *PromotionSpec `json:"spec"`
	// Status describes the current state of the transition represented by this
	// Promotion.
	Status PromotionStatus `json:"status,omitempty"`
}

// PromotionSpec describes the desired transition of a specific Environment into
// a specific EnvironmentState.
type PromotionSpec struct {
	// Environment specifies the name of the Environment to which this Promotion
	// applies. The Environment referenced by this field MUST be in the same
	// namespace as the Promotion.
	//
	// TODO: Use a webhook to make this immutable
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Environment string `json:"environment"`
	// State specifies the specific EnvironmentState into which the Environment
	// referenced by the Environment field should be transitioned. The State MUST
	// be among the Environment's Status.AvailableStates or the Promotion will
	// ultimately fail.
	//
	// TODO: Use a webhook to make this immutable
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

func init() {
	SchemeBuilder.Register(&Promotion{}, &PromotionList{})
}
