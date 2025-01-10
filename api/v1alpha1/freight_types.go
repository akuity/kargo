package v1alpha1

import (
	"crypto/sha1"
	"fmt"
	"path"
	"slices"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/helm"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=Alias,type=string,JSONPath=`.metadata.labels.kargo\.akuity\.io/alias`
// +kubebuilder:printcolumn:name=Origin (Kind),type=string,JSONPath=`.origin.kind`
// +kubebuilder:printcolumn:name=Origin (Name),type=string,JSONPath=`.origin.name`
// +kubebuilder:printcolumn:name=Age,type=date,JSONPath=`.metadata.creationTimestamp`

// Freight represents a collection of versioned artifacts.
type Freight struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Alias is a human-friendly alias for a piece of Freight. This is an optional
	// field. A defaulting webhook will sync this field with the value of the
	// kargo.akuity.io/alias label. When the alias label is not present or differs
	// from the value of this field, the defaulting webhook will set the label to
	// the value of this field. If the alias label is present and this field is
	// empty, the defaulting webhook will set the value of this field to the value
	// of the alias label. If this field is empty and the alias label is not
	// present, the defaulting webhook will choose an available alias and assign
	// it to both the field and label.
	Alias string `json:"alias,omitempty" protobuf:"bytes,7,opt,name=alias"`
	// Origin describes a kind of Freight in terms of its origin.
	//
	// +kubebuilder:validation:Required
	Origin FreightOrigin `json:"origin,omitempty" protobuf:"bytes,9,opt,name=origin"`
	// Commits describes specific Git repository commits.
	Commits []GitCommit `json:"commits,omitempty" protobuf:"bytes,3,rep,name=commits"`
	// Images describes specific versions of specific container images.
	Images []Image `json:"images,omitempty" protobuf:"bytes,4,rep,name=images"`
	// Charts describes specific versions of specific Helm charts.
	Charts []Chart `json:"charts,omitempty" protobuf:"bytes,5,rep,name=charts"`
	// Status describes the current status of this Freight.
	Status FreightStatus `json:"status,omitempty" protobuf:"bytes,6,opt,name=status"`
}

func (f *Freight) GetStatus() *FreightStatus {
	return &f.Status
}

// GenerateID deterministically calculates a piece of Freight's ID based on its
// contents and returns it.
func (f *Freight) GenerateID() string {
	size := len(f.Commits) + len(f.Images) + len(f.Charts)
	artifacts := make([]string, 0, size)
	for _, commit := range f.Commits {
		if commit.Tag != "" {
			// If we have a tag, incorporate it into the canonical representation of a
			// commit used when calculating Freight ID. This is necessary because one
			// commit could have multiple tags. Suppose we have already detected a
			// commit with a tag v1.0.0-rc.1 and produced the corresponding Freight.
			// Later, that same commit is tagged as v1.0.0. If we don't incorporate
			// the tag into the ID, we will never produce a new/distinct piece of
			// Freight for the new tag.
			artifacts = append(
				artifacts,
				fmt.Sprintf("%s:%s:%s", git.NormalizeURL(commit.RepoURL), commit.Tag, commit.ID),
			)
		} else {
			artifacts = append(
				artifacts,
				fmt.Sprintf("%s:%s", git.NormalizeURL(commit.RepoURL), commit.ID),
			)
		}
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
				path.Join(helm.NormalizeChartRepositoryURL(chart.RepoURL), chart.Name),
				chart.Version,
			),
		)
	}
	slices.Sort(artifacts)
	return fmt.Sprintf(
		"%x",
		sha1.Sum([]byte(
			fmt.Sprintf("%s:%s", f.Origin.String(), strings.Join(artifacts, "|")),
		)),
	)
}

// GitCommit describes a specific commit from a specific Git repository.
type GitCommit struct {
	// RepoURL is the URL of a Git repository.
	RepoURL string `json:"repoURL,omitempty" protobuf:"bytes,1,opt,name=repoURL"`
	// ID is the ID of a specific commit in the Git repository specified by
	// RepoURL.
	ID string `json:"id,omitempty" protobuf:"bytes,2,opt,name=id"`
	// Branch denotes the branch of the repository where this commit was found.
	Branch string `json:"branch,omitempty" protobuf:"bytes,3,opt,name=branch"`
	// Tag denotes a tag in the repository that matched selection criteria and
	// resolved to this commit.
	Tag string `json:"tag,omitempty" protobuf:"bytes,4,opt,name=tag"`
	// Message is the message associated with the commit. At present, this only
	// contains the first line (subject) of the commit message.
	Message string `json:"message,omitempty" protobuf:"bytes,6,opt,name=message"`
	// Author is the author of the commit.
	Author string `json:"author,omitempty" protobuf:"bytes,7,opt,name=author"`
	// Committer is the person who committed the commit.
	Committer string `json:"committer,omitempty" protobuf:"bytes,8,opt,name=committer"`
}

// DeepEquals returns a bool indicating whether the receiver deep-equals the
// provided GitCommit. I.e., all fields must be equal.
func (g *GitCommit) DeepEquals(other *GitCommit) bool {
	if g == nil && other == nil {
		return true
	}
	if g == nil || other == nil {
		return false
	}
	return g.RepoURL == other.RepoURL &&
		g.ID == other.ID &&
		g.Branch == other.Branch &&
		g.Tag == other.Tag &&
		g.Message == other.Message &&
		g.Author == other.Author &&
		g.Committer == other.Committer
}

// Equals returns a bool indicating whether two GitCommits are equivalent.
func (g *GitCommit) Equals(rhs *GitCommit) bool {
	if g == nil && rhs == nil {
		return true
	}
	if (g == nil && rhs != nil) || (g != nil && rhs == nil) {
		return false
	}
	// If we get to here, both operands are non-nil
	return g.RepoURL == rhs.RepoURL && g.ID == rhs.ID
}

// FreightStatus describes a piece of Freight's most recently observed state.
type FreightStatus struct {
	// CurrentlyIn describes the Stages in which this Freight is currently in use.
	CurrentlyIn map[string]CurrentStage `json:"currentlyIn,omitempty" protobuf:"bytes,3,rep,name=currentlyIn" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// VerifiedIn describes the Stages in which this Freight has been verified
	// through promotion and subsequent health checks.
	VerifiedIn map[string]VerifiedStage `json:"verifiedIn,omitempty" protobuf:"bytes,1,rep,name=verifiedIn" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// ApprovedFor describes the Stages for which this Freight has been approved
	// preemptively/manually by a user. This is useful for hotfixes, where one
	// might wish to promote a piece of Freight to a given Stage without
	// transiting the entire pipeline.
	ApprovedFor map[string]ApprovedStage `json:"approvedFor,omitempty" protobuf:"bytes,2,rep,name=approvedFor" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

// CurrentStage reflects a Stage's current use of Freight.
type CurrentStage struct {
	// Since is the time at which the Stage most recently started using the
	// Freight. This can be used to calculate how long the Freight has been in use
	// by the Stage.
	Since *metav1.Time `json:"since,omitempty" protobuf:"bytes,1,opt,name=since"`
}

// VerifiedStage describes a Stage in which Freight has been verified.
type VerifiedStage struct {
	// VerifiedAt is the time at which the Freight was verified in the Stage.
	VerifiedAt *metav1.Time `json:"verifiedAt,omitempty" protobuf:"bytes,1,opt,name=verifiedAt"`
	// LongestCompletedSoak represents the longest definite time interval wherein
	// the Freight was in CONTINUOUS use by the Stage. This value is updated as
	// Freight EXITS the Stage. If the Freight is currently in use by the Stage,
	// the time elapsed since the Freight ENTERED the Stage is its current soak
	// time, which may exceed the value of this field.
	LongestCompletedSoak *metav1.Duration `json:"longestSoak,omitempty" protobuf:"bytes,2,opt,name=longestSoak"`
}

// ApprovedStage describes a Stage for which Freight has been (manually)
// approved.
type ApprovedStage struct {
	// ApprovedAt is the time at which the Freight was approved for the Stage.
	ApprovedAt *metav1.Time `json:"approvedAt,omitempty" protobuf:"bytes,1,opt,name=approvedAt"`
}

// +kubebuilder:object:root=true

// FreightList is a list of Freight resources.
type FreightList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Freight `json:"items" protobuf:"bytes,2,rep,name=items"`
}
