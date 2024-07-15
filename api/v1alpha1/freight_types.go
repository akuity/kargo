package v1alpha1

import (
	"crypto/sha1"
	"fmt"
	"path"
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/helm"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=Alias,type=string,JSONPath=`.metadata.labels.kargo\.akuity\.io/alias`
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
	// Warehouse is the name of the Warehouse that created this Freight. This is a
	// required field. TODO: It is not clear yet how this field should be set in
	// the case of user-defined Freight.
	//
	// +kubebuilder:validation:Required
	Warehouse string `json:"warehouse,omitempty" protobuf:"bytes,8,opt,name=warehouse"`
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
	sort.Strings(artifacts)
	return fmt.Sprintf(
		"%x",
		sha1.Sum([]byte(strings.Join(artifacts, "|"))),
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
	// HealthCheckCommit is the ID of a specific commit. When specified,
	// assessments of Stage health will used this value (instead of ID) when
	// determining if applicable sources of Argo CD Application resources
	// associated with the Stage are or are not synced to this commit. Note that
	// there are cases (as in that of Kargo Render being utilized as a promotion
	// mechanism) wherein the value of this field may differ from the commit ID
	// found in the ID field.
	HealthCheckCommit string `json:"healthCheckCommit,omitempty" protobuf:"bytes,5,opt,name=healthCheckCommit"`
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
		g.HealthCheckCommit == other.HealthCheckCommit &&
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
	// VerifiedIn describes the Stages in which this Freight has been verified
	// through promotion and subsequent health checks.
	VerifiedIn map[string]VerifiedStage `json:"verifiedIn,omitempty" protobuf:"bytes,1,rep,name=verifiedIn" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	// ApprovedFor describes the Stages for which this Freight has been approved
	// preemptively/manually by a user. This is useful for hotfixes, where one
	// might wish to promote a piece of Freight to a given Stage without
	// transiting the entire pipeline.
	ApprovedFor map[string]ApprovedStage `json:"approvedFor,omitempty" protobuf:"bytes,2,rep,name=approvedFor" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

// VerifiedStage describes a Stage in which Freight has been verified.
type VerifiedStage struct{}

// ApprovedStage describes a Stage for which Freight has been (manually)
// approved.
type ApprovedStage struct{}

// +kubebuilder:object:root=true

// FreightList is a list of Freight resources.
type FreightList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Freight `json:"items" protobuf:"bytes,2,rep,name=items"`
}
