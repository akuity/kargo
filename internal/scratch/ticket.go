package scratch

// Ticket represents a change that is riding the proverbial rail line to
// production. For now, tickets are persisted to ConfigMaps.
//
// TODO: Replace this with a CRD.
type Ticket struct {
	// ID is a unique identifier for a Ticket.
	ID string `json:"id"`
	// Namespace is the Kubernetes namespace wherein the applicable Line for this
	// Ticket can be found.
	Namespace string `json:"namespace"`
	// Line is a reference to a K8sTA Line.
	Line string `json:"line"`
	// Source indicates how this ticket entered the system.
	Source string `json:"source"`
	// Change encapsulates the specific change this Ticket is meant to progress
	// through a series of environments.
	Change Change `json:"change"`
	// Status encapsulates the current state of the Ticket. e.g. How far it has
	// progressed.
	Status TicketStatus `json:"status"`
}

// Change is a description of a change that is being progressed through a series
// of environments by a Ticket.
type Change struct {
	// Type indicates a class of change that needs to be progressed through a
	// series of environments. The controller knows how to deal with different
	// classes of change based on the value of this field.
	Type string `json:"type"`
	// Image denotes a new image that is to be progressed through a series of
	// environments. The value of this field only has meaning when the value of
	// the Type field is "NewImage".
	Image string `json:"image"`
}

// TicketStatus represents the current status of a Ticket resource.
type TicketStatus struct {
	// TODO: Implement this. There should at least be enough information in here
	// to determine how far the Ticket has progressed and what the next logical
	// step for the controller should be.
}
