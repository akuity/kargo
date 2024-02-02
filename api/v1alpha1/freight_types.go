package v1alpha1

import (
	"crypto/sha1"
	"fmt"
	"path"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name=Alias,type=string,JSONPath=`.metadata.labels.kargo\.akuity\.io/alias`
//+kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// Freight represents a collection of versioned artifacts.
type Freight struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// ID is a system-assigned value that is derived deterministically from the
	// contents of the Freight. i.e. Two pieces of Freight can be compared for
	// equality by comparing their IDs.
	ID string `json:"id,omitempty"`
	// Commits describes specific Git repository commits.
	Commits []GitCommit `json:"commits,omitempty"`
	// Images describes specific versions of specific container images.
	Images []Image `json:"images,omitempty"`
	// Charts describes specific versions of specific Helm charts.
	Charts []Chart `json:"charts,omitempty"`
	// Status describes the current status of this Freight.
	Status FreightStatus `json:"status,omitempty"`
}

func (f *Freight) GetStatus() *FreightStatus {
	return &f.Status
}

// UpdateID deterministically calculates a piece of Freight's ID based on its
// contents and assigns it to the ID field.
func (f *Freight) UpdateID() {
	size := len(f.Commits) + len(f.Images) + len(f.Charts)
	artifacts := make([]string, 0, size)
	for _, commit := range f.Commits {
		artifacts = append(
			artifacts,
			fmt.Sprintf("%s:%s", commit.RepoURL, commit.ID),
		)
	}
	for _, image := range f.Images {
		artifacts = append(
			artifacts,
			// Note: This isn't the usual image representation using EITHER :<tag> OR @<digest>.
			// It is possible to have found an image with a tag that is already known, but with a
			// new digest -- as in the case of "mutable" tags like "latest". It is equally possible to
			// have found an image with a digest that is already known, but has been re-tagged.
			// To cover both cases, we incorporate BOTH tag and digest into the canonical
			// representation of an image used when calculating Freight ID.
			fmt.Sprintf("%s:%s@%s", image.RepoURL, image.Tag, image.Digest),
		)
	}
	for _, chart := range f.Charts {
		artifacts = append(
			artifacts,
			fmt.Sprintf(
				"%s:%s",
				// path.Join accounts for the possibility that chart.Name is empty
				path.Join(chart.RepoURL, chart.Name),
				chart.Version,
			),
		)
	}
	sort.Strings(artifacts)
	f.ID = fmt.Sprintf(
		"%x",
		sha1.Sum([]byte(strings.Join(artifacts, "|"))),
	)
}

// GitCommit describes a specific commit from a specific Git repository.
type GitCommit struct {
	// RepoURL is the URL of a Git repository.
	RepoURL string `json:"repoURL,omitempty"`
	// ID is the ID of a specific commit in the Git repository specified by
	// RepoURL.
	ID string `json:"id,omitempty"`
	// Branch denotes the branch of the repository where this commit was found.
	Branch string `json:"branch,omitempty"`
	// Tag denotes a tag in the repository that matched selection criteria and
	// resolved to this commit.
	Tag string `json:"tag,omitempty"`
	// HealthCheckCommit is the ID of a specific commit. When specified,
	// assessments of Stage health will used this value (instead of ID) when
	// determining if applicable sources of Argo CD Application resources
	// associated with the Stage are or are not synced to this commit. Note that
	// there are cases (as in that of Kargo Render being utilized as a promotion
	// mechanism) wherein the value of this field may differ from the commit ID
	// found in the ID field.
	HealthCheckCommit string `json:"healthCheckCommit,omitempty"`
	// Message is the git commit message
	Message string `json:"message,omitempty"`
	// Author is the git commit author
	Author string `json:"author,omitempty"`
}

// FreightStatus describes a piece of Freight's most recently observed state.
type FreightStatus struct {
	// VerifiedIn describes the Stages in which this Freight has been verified
	// through promotion and subsequent health checks.
	VerifiedIn map[string]VerifiedStage `json:"verifiedIn,omitempty"`
	// ApprovedFor describes the Stages for which this Freight has been approved
	// preemptively/manually by a user. This is useful for hotfixes, where one
	// might wish to promote a piece of Freight to a given Stage without
	// transiting the entire pipeline.
	ApprovedFor map[string]ApprovedStage `json:"approvedFor,omitempty"`
}

// VerifiedStage describes a Stage in which Freight has been verified.
type VerifiedStage struct{}

// ApprovedStage describes a Stage for which Freight has been (manually)
// approved.
type ApprovedStage struct{}

//+kubebuilder:object:root=true

// FreightList is a list of Freight resources.
type FreightList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Freight `json:"items"`
}
