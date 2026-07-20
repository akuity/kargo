package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
type Role struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	KargoManaged      bool                `json:"kargoManaged,omitempty"`
	Claims            []Claim             `json:"claims,omitempty"`
	Rules             []rbacv1.PolicyRule `json:"rules,omitempty"`
}

// +kubebuilder:object:root=true
type RoleResources struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	ServiceAccount    corev1.ServiceAccount `json:"serviceAccount,omitempty"`
	Roles             []rbacv1.Role         `json:"roles,omitempty"`
	ClusterRoles      []rbacv1.ClusterRole  `json:"clusterRoles,omitempty"`
	RoleBindings      []rbacv1.RoleBinding  `json:"roleBindings,omitempty"`
}

type ResourceDetails struct {
	ResourceType string   `json:"resourceType,omitempty"`
	ResourceName string   `json:"resourceName,omitempty"`
	Verbs        []string `json:"verbs,omitempty"`
} // @name ResourceDetails

type Claim struct {
	Name   string   `json:"name,omitempty"`
	Values []string `json:"values,omitempty"`
} // @name Claim

type ServiceAccountReference struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}
