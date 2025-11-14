package v1alpha1

import (
	"encoding/json"
	"fmt"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// Metadata is a map of arbitrary metadata associated with the Freight.
	// This is useful for storing additional information about the Freight
	// or Promotion that can be shared across steps or stages.
	Metadata map[string]apiextensionsv1.JSON `json:"metadata,omitempty" protobuf:"bytes,4,rep,name=metadata" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

// IsCurrentlyIn returns whether the Freight is currently in the specified
// Stage.
func (f *Freight) IsCurrentlyIn(stage string) bool {
	// NB: This method exists for convenience. It doesn't require the caller to
	// know anything about the Freight status' internal data structure.
	_, in := f.Status.CurrentlyIn[stage]
	return in
}

// IsVerifiedIn returns whether the Freight has been verified in the specified
// Stage.
func (f *Freight) IsVerifiedIn(stage string) bool {
	// NB: This method exists for convenience. It doesn't require the caller to
	// know anything about the Freight status' internal data structure.
	_, verified := f.Status.VerifiedIn[stage]
	return verified
}

// IsApprovedFor returns whether the Freight has been approved for the specified
// Stage.
func (f *Freight) IsApprovedFor(stage string) bool {
	// NB: This method exists for convenience. It doesn't require the caller to
	// know anything about the Freight status' internal data structure.
	_, approved := f.Status.ApprovedFor[stage]
	return approved
}

// GetLongestSoak returns the longest soak time for the Freight in the specified
// Stage if it's been verified in that Stage. If it has not, zero will be
// returned instead. If the Freight is currently in use by the specified Stage,
// the current soak time is calculated and compared to the longest completed
// soak time on record.
func (f *Freight) GetLongestSoak(stage string) time.Duration {
	if _, verified := f.Status.VerifiedIn[stage]; !verified {
		return 0
	}
	var longestCompleted time.Duration
	if record, isVerified := f.Status.VerifiedIn[stage]; isVerified && record.LongestCompletedSoak != nil {
		longestCompleted = record.LongestCompletedSoak.Duration
	}
	var current time.Duration
	if record, isCurrent := f.Status.CurrentlyIn[stage]; isCurrent {
		current = time.Since(record.Since.Time)
	}
	return time.Duration(max(longestCompleted.Nanoseconds(), current.Nanoseconds()))
}

// HasSoakedIn returns whether the Freight has soaked in the specified Stage for
// at least the specified duration. If the specified duration is nil, this
// method will return true.
func (f *Freight) HasSoakedIn(stage string, dur *metav1.Duration) bool {
	if f == nil {
		return false
	}
	if dur == nil {
		return true
	}
	return f.GetLongestSoak(stage) >= dur.Duration
}

// AddCurrentStage updates the Freight status to reflect that the Freight is
// currently in the specified Stage.
func (f *FreightStatus) AddCurrentStage(stage string, since time.Time) {
	if _, alreadyIn := f.CurrentlyIn[stage]; !alreadyIn {
		if f.CurrentlyIn == nil {
			f.CurrentlyIn = make(map[string]CurrentStage)
		}
		f.CurrentlyIn[stage] = CurrentStage{
			Since: &metav1.Time{Time: since},
		}
	}
}

// RemoveCurrentStage updates the Freight status to reflect that the Freight is
// no longer in the specified Stage. If the Freight was verified in the
// specified Stage, the longest completed soak time will be updated if
// necessary.
func (f *FreightStatus) RemoveCurrentStage(stage string) {
	if record, in := f.CurrentlyIn[stage]; in {
		if record.Since != nil {
			soak := time.Since(record.Since.Time)
			if vi, verified := f.VerifiedIn[stage]; verified {
				if vi.LongestCompletedSoak == nil || soak > vi.LongestCompletedSoak.Duration {
					vi.LongestCompletedSoak = &metav1.Duration{Duration: soak}
					f.VerifiedIn[stage] = vi
				}
			}
		}
		delete(f.CurrentlyIn, stage)
	}
}

// AddVerifiedStage updates the Freight status to reflect that the Freight has
// been verified in the specified Stage.
func (f *FreightStatus) AddVerifiedStage(stage string, verifiedAt time.Time) {
	if _, verified := f.VerifiedIn[stage]; !verified {
		record := VerifiedStage{VerifiedAt: &metav1.Time{Time: verifiedAt}}
		if f.VerifiedIn == nil {
			f.VerifiedIn = map[string]VerifiedStage{stage: record}
		}
		f.VerifiedIn[stage] = record
	}
}

// AddApprovedStage updates the Freight status to reflect that the Freight has
// been approved for the specified Stage.
func (f *FreightStatus) AddApprovedStage(stage string, approvedAt time.Time) {
	if _, approved := f.ApprovedFor[stage]; !approved {
		record := ApprovedStage{ApprovedAt: &metav1.Time{Time: approvedAt}}
		if f.ApprovedFor == nil {
			f.ApprovedFor = map[string]ApprovedStage{stage: record}
		}
		f.ApprovedFor[stage] = record
	}
}

// UpsertMetadata inserts or updates the given key in Freight status Metadata
func (f *FreightStatus) UpsertMetadata(key string, data any) error {
	if len(f.Metadata) == 0 {
		f.Metadata = make(map[string]apiextensionsv1.JSON)
	}

	if key == "" {
		return fmt.Errorf("key must not be empty")
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	f.Metadata[key] = apiextensionsv1.JSON{
		Raw: dataBytes,
	}
	return nil
}

// GetMetadata retrieves the data associated with the given key from Freight status Metadata
func (f *FreightStatus) GetMetadata(key string, data any) (bool, error) {
	dataBytes, ok := f.Metadata[key]

	if !ok {
		return false, nil
	}

	if err := json.Unmarshal(dataBytes.Raw, data); err != nil {
		return false, err
	}
	return true, nil
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
