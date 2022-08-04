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
	// Environments enumerates a logical ordering of environments through which a
	// change represented by a Ticket should be progressed.
	Environments []Environment `json:"environments,omitempty"`
}

// Environment represents a single environment through which a change
// represented by a Ticket should be progressed.
type Environment struct {
	// Name is a name for the environment.
	Name string `json:"name,omitempty"`
	// ArgoCDApplication is a reference to an existing Argo CD Application
	// resource that managed deployments to this Environment.
	Application string `json:"application,omitempty"`
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
