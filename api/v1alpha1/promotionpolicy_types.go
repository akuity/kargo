package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum={Group,Role,ServiceAccount,User}
type AuthorizedPromoterSubjectType string

const (
	AuthorizedPromoterSubjectTypeGroup          AuthorizedPromoterSubjectType = "Group"
	AuthorizedPromoterSubjectTypeRole           AuthorizedPromoterSubjectType = "Role"
	AuthorizedPromoterSubjectTypeServiceAccount AuthorizedPromoterSubjectType = "ServiceAccount"
	AuthorizedPromoterSubjectTypeUser           AuthorizedPromoterSubjectType = "User"
)

//+kubebuilder:resource:shortName={promopolicy,promopolicies}
//+kubebuilder:object:root=true

// PromotionPolicy provides fine-grained access control beyond what Kubernetes
// RBAC is capable of. A PromotionPolicy names an Environment and enumerates
// subjects (such as users, groups, ServiceAccounts, or RBAC Roles) that are
// authorized to create Promotions for that Environment. It is through
// PromotionPolicies that multiple users may be permitted to create Promotion
// resources in a given namespace, but creation of Promotion resources for
// specific Environments may be restricted.
type PromotionPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Environment references an Environment in the same namespace as this
	// PromotionPolicy to which this PromotionPolicy applies.
	//
	//+kubebuilder:validation:MinLength=1
	//+kubebuilder:validation:Pattern=^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$
	Environment string `json:"environment"`
	// AuthorizedPromoters enumerates subjects (such as users, groups,
	// ServiceAccounts, or RBAC Roles) that are authorized to create Promotions
	// for the Environment referenced by the Environment field.
	AuthorizedPromoters []AuthorizedPromoter `json:"authorizedPromoters,omitempty"`
	// EnableAutoPromotion indicates whether new EnvironmentStates can
	// automatically be promoted into the Environment referenced by the
	// Environment field. Note: There are other conditions also required for an
	// auto-promotion to occur. Specifically, there must be a single source of new
	// EnvironmentStates, so regardless of the value of this field, an
	// auto-promotion could never occur for an Environment subscribed to MULTIPLE
	// upstream environments. This field defaults to false, but is commonly set to
	// true for Environments that subscribe to repositories instead of other,
	// upstream Environments. This allows users to define Environments that are
	// automatically updated as soon as new materials are detected.
	EnableAutoPromotion bool `json:"enableAutoPromotion,omitempty"`
}

// AuthorizedPromoter identifies a single subject that is authorized to create
// Promotion resources referencing a particular Environment.
type AuthorizedPromoter struct {
	// SubjectType identifies the type of subject being authorized to create
	// Promotion resources referencing a particular Environment.
	//
	//+kubebuilder:validation:Required
	SubjectType AuthorizedPromoterSubjectType `json:"subjectType"`
	// Name is the name of a subject authorized to create Promotion
	// resources referencing a particular Environment. This could be a username,
	// group name, ServiceAccount name, or Role name.
	//
	//+kubebuilder:validation:Required
	Name string `json:"name"`
}

//+kubebuilder:object:root=true

// PromotionPolicyList contains a list of PromotionPolicies
type PromotionPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PromotionPolicy `json:"items"`
}
