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
//+kubebuilder:subresource:status

// Track is the Schema for the tracks API
type Track struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Spec encapsulates the desired state of a Track.
	Spec TrackSpec `json:"spec,omitempty"`
	// Status encapsulates the status of the Track.
	Status TrackStatus `json:"status,omitempty"`
}

// TrackSpec encapsulates the desired state of a Track.
type TrackSpec struct {
	// Disabled indicates whether this Track is disabled.
	Disabled bool `json:"disabled,omitempty"`
	// ImageRepositorySubscriptions specifies image repositories that the Track is
	// subscribed to. When a push to any one of these repositories is
	// detected, a new Ticket will be created to effect progression of the new
	// image through the Track's Stations.
	ImageRepositorySubscriptions []ImageRepositorySubscription `json:"imageRepositorySubscriptions,omitempty"` // nolint: lll
	// GitRepositorySubscription specifies a git repository that the Track is
	// subscribed to. When changes are detected in the source branch of that
	// repository and they are deemed to affect base configuration (all
	// environments), a new Ticket will be created to effect progression of the
	// change through the Track's Stations.
	GitRepositorySubscription *GitRepositorySubscription `json:"gitRepositorySubscription,omitempty"` // nolint: lll
	// ConfigManagement encapsulates details of which configuration management
	// tool is to be used for this Track and, if applicable, configuration options
	// for the selected tool.
	ConfigManagement ConfigManagementConfig `json:"configManagement,omitempty"`
	// Stations enumerates points along the Track through which a change
	// represented by a Ticket may be progressed.
	Stations []Station `json:"stations,omitempty"`
}

// ImageRepositorySubscription defines a subscription to an image repository.
type ImageRepositorySubscription struct {
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

// GitRepositorySubscription defines a subscription to a git repository.
type GitRepositorySubscription struct {
	// RepoURL specifies the URL of the git repository to subscribe to.
	RepoURL string `json:"repoURL,omitempty"`
}

// ConfigManagementConfig is a wrapper around more specific configuration for
// one of three supported configuration management tools: helm, kustomize, or
// ytt. Only one of its fields may be non-nil. Compare this to how a
// EnvVarSource or VolumeSource works in the core Kubernetes API.
type ConfigManagementConfig struct {
	// Helm encapsulates optional Helm configuration options.
	Helm *HelmConfig `json:"helm,omitempty"`
	// Kustomize encapsulates optional Kustomize configuration options.
	Kustomize *KustomizeConfig `json:"kustomize,omitempty"`
	// Ytt encapsulates optional ytt configuration options.
	Ytt *YttConfig `json:"ytt,omitempty"`
}

// HelmConfig encapsulates optional Helm configuration options.
type HelmConfig struct {
	// ReleaseName specified the release name that will be used when running
	// `helm template <release name> <chart> --values <values>`
	ReleaseName string `json:"releaseName,omitempty"`
}

// KustomizeConfig encapsulates optional Kustomize configuration options.
type KustomizeConfig struct {
}

// YttConfig encapsulates optional ytt configuration options.
type YttConfig struct {
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

// TrackStatus encapsulates the status of the Track.
type TrackStatus struct {
	// GitSyncStatus encapsulates the details of what point in a repository was
	// last synced to and when.
	GitSyncStatus *GitSyncStatus `json:"gitSyncStatus,omitempty"`
}

// GitSyncStatus encapsulates the details of what point in a repository was last
// synced to and when.
type GitSyncStatus struct {
	// Commit is the commit ID (sha) found at HEAD at the time of the last sync.
	Commit string `json:"commit,omitempty"`
	// Time is the time of the most recent sync.
	Time *metav1.Time `json:"time,omitempty"`
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
