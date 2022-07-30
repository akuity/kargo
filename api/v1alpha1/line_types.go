package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true

// Line is the Schema for the lines API
type Line struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ImageRepositories specifies image repositories that the Line is effectively
	// subscribed to. When a push to any one of these repositories is detected, it
	// will trigger the progressive deployment of the new image through the Line's
	// environments.
	ImageRepositories []string `json:"imageRepositories"`
	// Environments enumerates a logical ordering of environments, each
	// represented by a string reference to an (assumed to be) existing Argo CD
	// Application resource in the Kubernetes namespace indicated by the Namespace
	// field.
	Environments []string `json:"environments"`
}

//+kubebuilder:object:root=true

// LineList contains a list of Line
type LineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Line `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Line{}, &LineList{})
}
