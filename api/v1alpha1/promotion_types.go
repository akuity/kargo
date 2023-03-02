package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&Promotion{}, &PromotionList{})
}

//+kubebuilder:resource:shortName={promo,promos}
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

type Promotion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PromotionSpec   `json:"spec,omitempty"`
	Status            PromotionStatus `json:"status,omitempty"`
}

type PromotionSpec struct{}

type PromotionStatus struct {
	// Error describes any errors that have occurred during Promotion
	// reconciliation.
	Error string `json:"error,omitempty"`
}

//+kubebuilder:object:root=true

// PromotionList contains a list of Promotion
type PromotionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Promotion `json:"items"`
}
