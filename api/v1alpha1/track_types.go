package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdateStrategy represents a strategy for determining when one tag of a given
// image is newer than another, thus making it eligible to be promoted through a
// succession of Stations.
type UpdateStrategy string

const (
	// UpdateStrategyDigest specifies a strategy of an updating an image to the to
	// the most recently pushed version of a mutable tag.
	UpdateStrategyDigest UpdateStrategy = "Digest"
	// UpdateStrategyLatest specifies a strategy of updating an image to the tag
	// with the most recent creation date.
	UpdateStrategyLatest UpdateStrategy = "Latest"
	// UpdateStrategyName specifies a strategy of updating an image to the tag
	// with the latest entry from an alphabetically sorted list.
	UpdateStrategyName UpdateStrategy = "Name"
	// UpdateStrategySemver specifies a strategy of updating an image to the tag
	// with the highest allowed semantic version.
	UpdateStrategySemver UpdateStrategy = "Semver"
)

//+kubebuilder:object:root=true

// Track is the Schema for the tracks API
type Track struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Disabled indicates whether this Track is disabled.
	Disabled bool `json:"disabled,omitempty"`
	// RepositorySubscriptions specifies image repositories that the Track is
	// effectively subscribed to. When a push to any one of these repositories is
	// detected, it will trigger the progressive deployment of the new image
	// through the Track's Stations.
	RepositorySubscriptions []RepositorySubscription `json:"repositorySubscriptions,omitempty"` // nolint: lll
	// Stations enumerates points along the Track through which a change
	// represented by a Ticket may be progressed.
	Stations []Station `json:"stations,omitempty"`
}

// RepositorySubscription defines a subscription to an image repository.
type RepositorySubscription struct {
	// RepoURL specifies the URL of the image repository to subscribe to. The
	// value in this field MUST NOT include an image tag.
	RepoURL string `json:"repoURL,omitempty"`
	// UpdateStrategy specifies the rules for how to identify the newest version
	// of the image specified by the RepoURL field.
	UpdateStrategy UpdateStrategy `json:"updateStrategy,omitempty"`
	// AllowTags is a regular expression that can optionally be used to limit the
	// image tags that are considered in determining the newest version of an
	// image.
	AllowTags string `json:"allowTags,omitempty"`
	// IgnoreTags is a list of tags that must be ignored when determining the
	// newest version of an image. No regular expressions or glob patterns are
	// supported yet.
	IgnoreTags []string `json:"ignoreTags,omitempty"`
	// PullSecret is a reference to a Kubernetes Secret containing repository
	// credentials. If left unspecified, K8sTA will fall back on globally
	// configured repository credentials, if they exist.
	PullSecret string `json:"pullSecret,omitempty"`
}

// Station represents a single point on a Track through which a change
// represented by a Ticket may be progressed.
type Station struct {
	// Name is a name for the Station.
	Name string `json:"name,omitempty"`
	// Disabled indicates whether this Station is disabled and effectively
	// removed from the Track.
	Disabled bool `json:"disabled,omitempty"`
	// Applications is a list of references to existing Argo CD Applications.
	// Progressing through the Station is effected via deployment of each of these
	// Applications.
	Applications []ApplicationReference `json:"applications,omitempty"`
	// Tracks is a list of references to other existing K8sTA Track resources.
	// When the change represented by a Ticket reaches this Station, a new Ticket
	// representing the same change will be created for each of these Tracks. i.e.
	// This permits the composition of complex tracks from segments of linear
	// Track.
	Tracks []TrackReference `json:"tracks,omitempty"`
}

// ApplicationReference is a reference to an existing Argo CD Application.
type ApplicationReference struct {
	// Name is the name of an existing Argo CD Application.
	Name string `json:"name,omitempty"`
	// Disabled indicates whether deployments to the referenced Argo CD
	// Application should be bypassed as changes progress along the Track.
	Disabled bool `json:"disabled,omitempty"`
}

// TrackReference is a reference to a Track.
type TrackReference struct {
	// Name is the name of an existing Track.
	Name string `json:"name,omitempty"`
	// Disabled indicates whether the junction represented by this TrackReference
	// should be ignored as changes progress along the Track making the reference.
	Disabled bool `json:"disabled,omitempty"`
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
