package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:resource:shortName={promopolicy,promopolicies}
//+kubebuilder:object:root=true

// PromotionPolicy specifies whether a given Stage is eligible for
// auto-promotion to newly discovered StageStates.
type PromotionPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Stage references a Stage in the same project as this PromotionPolicy to
	// which this PromotionPolicy applies.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Stage string `json:"stage,"`
	// EnableAutoPromotion indicates whether new StageStates can automatically be
	// promoted into the Stage referenced by the Stage field. Note: There are
	// other conditions also required for an auto-promotion to occur.
	// Specifically, there must be a single source of new StageStates, so
	// regardless of the value of this field, an auto-promotion could never occur
	// for a Stage subscribed to MULTIPLE upstream Stages. This field defaults to
	// false, but is commonly set to true for Stages that subscribe to
	// repositories instead of other, upstream Stages. This allows users to define
	// Stages that are automatically updated as soon as new materials are
	// detected.
	EnableAutoPromotion bool `json:"enableAutoPromotion,omitempty"`
}

//+kubebuilder:object:root=true

// PromotionPolicyList contains a list of PromotionPolicies
type PromotionPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PromotionPolicy `json:"items"`
}
