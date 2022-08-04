package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true

// Track is the Schema for the tracks API
type Track struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ImageRepositories specifies image repositories that the Track is
	// effectively subscribed to. When a push to any one of these repositories is
	// detected, it will trigger the progressive deployment of the new image
	// through the Track's environments.
	ImageRepositories []string `json:"imageRepositories"`
	// Environments enumerates a logical ordering of environments, each
	// represented by a string reference to an (assumed to be) existing Argo CD
	// Application resource in the Kubernetes namespace indicated by the Namespace
	// field.
	Environments []string `json:"environments"`
}

//+kubebuilder:object:root=true

// TrackList contains a list of Track
type TrackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Track `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Track{}, &TrackList{})
}
