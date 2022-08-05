package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TicketState is a type representing the current state of a Ticket.
type TicketState string

const (
	// TicketStateCompleted is a constant representing a Ticket that has
	// progressed all the way to the last environment in the associated Track.
	TicketStateCompleted TicketState = "Completed"
	// TicketStateFailed is a constant representing a Ticket that can not be
	// progressed further for whatever reason.
	TicketStateFailed TicketState = "Failed"
	// TicketStateNew is a constant representing a brand new Ticket. Nothing has
	// been done yet to address the change represented by the Ticket.
	TicketStateNew TicketState = "New"
	// TicketStateProgressing is a constant representing a Ticket whose change
	// is already being progressed through a series of environments.
	TicketStateProgressing TicketState = "Progressing"
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
	Migration *Migration `json:"migration,omitempty"`
}

// Migration represents a bit of a Ticket's progress along a Track involving
// migration from one environment to another.
type Migration struct {
	// TargetEnvironment indicates the environment into which this Transition aims
	// to migrate the change represented by the Ticket.
	TargetEnvironment string `json:"targetEnvironment,omitempty"`
	// Commits records all git commits made to effect an update to
	// TargetEnvironment.
	Commits []Commit `json:"commits,omitempty"`
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
	// through a series of environments.
	Change Change `json:"change,omitempty"`
	// Status encapsulates the status of the Ticket.
	Status TicketStatus `json:"status,omitempty"`
}

// Change describes a change that is to be progressed through a series of
// environments by a Ticket. Only one of its fields may be non-nil. Compare this
// to how a EnvVarSource or VolumeSource works in the core Kubernetes API.
type Change struct {
	// NewImage encapsulates the details of a new image that is to be progressed
	// through a series of environments.
	NewImage *NewImageChange `json:"newImage,omitempty"`
}

type NewImageChange struct {
	// Repo denotes the image (without tag) that is to be progressed through a
	// series of environments.
	Repo string `json:"imageRepo,omitempty"`
	// Tag qualifies which image from the repository specified by the Repo field
	// is to be progressed through a series of environments.
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
