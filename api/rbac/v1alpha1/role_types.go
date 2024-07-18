package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
type Role struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	KargoManaged      bool                `json:"kargoManaged,omitempty" protobuf:"varint,2,opt,name=kargoManaged"`
	Claims            []Claim             `json:"claims,omitempty" protobuf:"bytes,3,rep,name=claims"`
	Rules             []rbacv1.PolicyRule `json:"rules,omitempty" protobuf:"bytes,6,rep,name=rules"`
}

// +kubebuilder:object:root=true
type RoleResources struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	ServiceAccount    corev1.ServiceAccount `json:"serviceAccount,omitempty" protobuf:"bytes,2,opt,name=serviceAccount"`
	Roles             []rbacv1.Role         `json:"roles,omitempty" protobuf:"bytes,3,rep,name=roles"`
	RoleBindings      []rbacv1.RoleBinding  `json:"roleBindings,omitempty" protobuf:"bytes,4,rep,name=roleBindings"`
}

type ResourceDetails struct {
	ResourceType string   `json:"resourceType,omitempty" protobuf:"bytes,1,opt,name=resourceType"`
	ResourceName string   `json:"resourceName,omitempty" protobuf:"bytes,2,opt,name=resourceName"`
	Verbs        []string `json:"verbs,omitempty" protobuf:"bytes,3,rep,name=verbs"`
}

// type UserClaims struct {
// 	Subs   []string `json:"subs,omitempty" protobuf:"bytes,1,rep,name=subs"`
// 	Emails []string `json:"emails,omitempty" protobuf:"bytes,2,rep,name=emails"`
// 	Groups []string `json:"groups,omitempty" protobuf:"bytes,3,rep,name=groups"`
// }

type Claim struct {
	Name   string
	Values []string
}

type UserClaims struct {
	Claims []Claim `json:"claims,omitempty" protobuf:"bytes,1,rep,name=claims"`
}
