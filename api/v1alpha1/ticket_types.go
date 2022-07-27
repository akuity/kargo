package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ChangeType is a type representing different types of Changes meant to be
// progressed through a series of environments.
type ChangeType string

const (
	// ChangeTypeNewImage is a constant representing a Change involving a new
	// image that is to be progressed through a series of environments.
	ChangeTypeNewImage ChangeType = "NewImage"
)

// TicketState is a type representing the current state of a Ticket.
type TicketState string

const (
	// TicketStateCompleted is a constant representing a Ticket that has
	// progressed all the way to the last environment in the associated Line.
	TicketStateCompleted TicketState = "Completed"
	// TicketStateFailed is a constant representing a Ticket that can not be
	// progressed further for whatever reason.
	TicketStateFailed TicketState = "FAILED"
	// TicketStateNew is a constant representing a brand new Ticket. Nothing has
	// been done yet to address the change represented by the Ticket.
	TicketStateNew TicketState = "NEW"
	// TicketStateProgressing is a constant representing a Ticket whose change
	// is already being progressed through a series of environments.
	TicketStateProgressing TicketState = "PROGRESSING"
)

// TicketSpec defines the desired state of Ticket
type TicketSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Line is a reference to a K8sTA Line.
	Line string `json:"line,omitempty"`
	// Source indicates how this ticket entered the system.
	Source string `json:"source,omitempty"`
	// Change encapsulates the specific change this Ticket is meant to progress
	// through a series of environments.
	Change Change `json:"change,omitempty"`
}

// Change is a description of a change that is being progressed through a series
// of environments by a Ticket.
type Change struct {
	// Type indicates a class of change that needs to be progressed through a
	// series of environments. The controller knows how to deal with different
	// classes of change based on the value of this field.
	Type ChangeType `json:"type,omitempty"`
	// ImageRepo denotes a new image (without tag) that is to be progressed
	// through a series of environments. The value of this field only has meaning
	// when the value of the Type field is "NewImage".
	ImageRepo string `json:"imageRepo,omitempty"`
	// ImageTag qualifies which image from the repository specified by the
	// ImageRepo filed is to be progressed through a series of environments. The
	// value of this field only has meaning when the value of the Type field is
	// "NewImage".
	ImageTag string `json:"imageTag,omitempty"`
}

// TicketStatus defines the observed state of Ticket
type TicketStatus struct {
	// TicketState represents the overall state of the Ticket.
	State TicketState `json:"state,omitempty"`
	// StateReason provides context for why the Ticket is in the State that it is.
	StateReason string `json:"stateReason,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Ticket is the Schema for the tickets API
type Ticket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TicketSpec   `json:"spec,omitempty"`
	Status TicketStatus `json:"status,omitempty"`
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
