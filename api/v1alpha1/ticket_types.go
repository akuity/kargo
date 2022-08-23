package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TicketState is a type representing the current state of a Ticket.
type TicketState string

const (
	// TicketStateCompleted is a constant representing a Ticket that has
	// progressed all the way to the last Station in the associated Track.
	TicketStateCompleted TicketState = "Completed"
	// TicketStateFailed is a constant representing a Ticket that can not be
	// progressed further for whatever reason.
	TicketStateFailed TicketState = "Failed"
	// TicketStateNew is a constant representing a brand new Ticket. Nothing has
	// been done yet to address the change represented by the Ticket.
	TicketStateNew TicketState = "New"
	// TicketStateProgressing is a constant representing a Ticket whose change
	// is already being progressed through a series of Stations.
	TicketStateProgressing TicketState = "Progressing"
	// TicketStateSuspended is a constant representing a Ticket whose change was
	// being progressed through a series of Stations, but has stalled, probably
	// temporarily, because all Argo CD applications that the change is currently
	// being applied to are themselves in a Suspended state..
	TicketStateSuspended TicketState = "Suspended"
)

// TicketStatus defines the observed state of Ticket
type TicketStatus struct {
	// TicketState represents the overall state of the Ticket.
	State TicketState `json:"state,omitempty"`
	// StateReason provides context for why the Ticket is in the State that it is.
	StateReason string `json:"stateReason,omitempty"`
	// Progress
	Progress []ProgressRecord `json:"progress,omitempty"`
}

// ProgressRecord records a single bit of a Ticket's progress along its Track.
// Only one of its fields may be non-nil. Compare this to how a EnvVarSource or
// VolumeSource works in the core Kubernetes API.
type ProgressRecord struct {
	// Migration represents a bit of a Ticket's progress along a Track involving
	// migration from one Station to another.
	Migration *Migration `json:"migration,omitempty"`
	// SkippedStation represents a bit of a Ticket's progress along a Track
	// involving a Station that was bypassed because it was disabled.
	SkippedStation string `json:"skippedStation,omitempty"`
}

// Migration represents a bit of a Ticket's progress along a Track involving
// migration from one Station to another.
type Migration struct {
	// TargetStation indicates the Station on the associated Track into which this
	// Migration aims to migrate the change represented by the Ticket.
	TargetStation string `json:"targetStation,omitempty"`
	// Commits records all git commits made to effect the migration to
	// TargetStation.
	Commits []Commit `json:"commits,omitempty"`
	// SkippedApplications lists Applications referenced by the TargetStation that
	// were bypassed during this Migration because they were disabled.
	SkippedApplications []string `json:"skippedApplications,omitempty"`
	// SkippedTracks lists Tracks referenced by the TargetStation that were
	// bypassed during this Migration because they were disabled.
	SkippedTracks []string `json:"skippedTracks,omitempty"`
	// Tickets is a list of references to Ticket resources created to progress the
	// same change represented by this Ticket down another Track.
	Tickets []TicketReference `json:"tickets,omitempty"`
	// Started indicates the time the migration started.
	Started *metav1.Time `json:"started,omitempty"`
	// Completed indicates the time the migration completed. A nil value means
	// the migration is not yet complete.
	Completed *metav1.Time `json:"completed,omitempty"`
}

// Commit represents a git commit made with the intent to effect an update to
// a specific Argo CD Application.
type Commit struct {
	// TargetApplication records the Argo CD Application resource for which this
	// Commit was intended to effect an update.
	TargetApplication string `json:"targetApplication,omitempty"`
	// SHA records the ID of the git commit.
	SHA string `json:"sha,omitempty"`
}

// TicketReference is a reference to a Ticket.
type TicketReference struct {
	// Name is the name of a Ticket.
	Name string `json:"name,omitempty"`
	// Track is the name of the Track this Ticket references. This information is
	// duplicated here for operator convenience.
	Track string `json:"track,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Ticket is the Schema for the tickets API
type Ticket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Track is a reference to a K8sTA Track.
	Track string `json:"track,omitempty"`
	// Source indicates how this ticket entered the system.
	Source string `json:"source,omitempty"`
	// Change encapsulates the specific change this Ticket is meant to progress
	// through a series of Stations.
	Change Change `json:"change,omitempty"`
	// Status encapsulates the status of the Ticket.
	Status TicketStatus `json:"status,omitempty"`
}

// Change describes a change that is to be progressed through a series of
// Stations by a Ticket. Only one of its fields may be non-nil. Compare this
// to how a EnvVarSource or VolumeSource works in the core Kubernetes API.
type Change struct {
	// NewImage encapsulates the details of one or more new images that are to be
	// progressed through a series of Stations.
	NewImages *NewImagesChange `json:"newImages,omitempty"`
}

// NewImagesChanges encapsulates the details of one or more new images that are
// to be progressed through a series of Stations.
type NewImagesChange struct {
	Images []Image `json:"images,omitempty"`
}

// Image encapsulates the details of a single image.
type Image struct {
	// Repo denotes the image (without tag) that is to be progressed through a
	// series of Stations.
	Repo string `json:"imageRepo,omitempty"`
	// Tag qualifies which image from the repository specified by the Repo field
	// is to be progressed through a series of Stations.
	Tag string `json:"imageTag,omitempty"`
}

//+kubebuilder:object:root=true

// TicketList contains a list of Ticket
type TicketList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Ticket `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Ticket{}, &TicketList{})
}
