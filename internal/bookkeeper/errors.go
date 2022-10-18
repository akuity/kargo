package bookkeeper

import "fmt"

// ErrBadRequest represents an error wherein an invalid request has been
// rejected by the Bookkeeper server.
type ErrBadRequest struct {
	// Reason is a natural language explanation for why the request is invalid.
	Reason string `json:"reason,omitempty"`
	// Details may further qualify why a request is invalid. For instance, if
	// the Reason field states that request validation failed, the Details field,
	// may enumerate specific request schema violations.
	Details []string `json:"details,omitempty"`
}

func (e *ErrBadRequest) Error() string {
	if len(e.Details) == 0 {
		return fmt.Sprintf("Bad request: %s", e.Reason)
	}
	msg := fmt.Sprintf("Bad request: %s:", e.Reason)
	for i, detail := range e.Details {
		msg = fmt.Sprintf("%s\n  %d. %s", msg, i, detail)
	}
	return msg
}

// ErrNotFound represents an error wherein a resource presumed to exist could
// not be located.
type ErrNotFound struct {
	// Type identifies the type of the resource that could not be located.
	Type string `json:"type,omitempty"`
	// ID is the identifier of the resource of type Type that could not be
	// located.
	ID string `json:"id,omitempty"`
	// Reason is a natural language explanation around why the resource could not
	// be located.
	Reason string `json:"reason,omitempty"`
}

func (e *ErrNotFound) Error() string {
	if e.Type == "" && e.ID == "" && e.Reason != "" {
		return e.Reason
	}

	msg := fmt.Sprintf("%s %q not found", e.Type, e.ID)
	if e.Reason != "" {
		return msg + fmt.Sprintf(": %s", e.Reason)
	}
	return msg + "."
}

// ErrConflict represents an error wherein a request cannot be completed because
// it would violate some constraint of the system, for instance creating a new
// resource with an identifier already used by another resource of the same
// type.
type ErrConflict struct {
	// Type identifies the type of the resource that the conflict applies to.
	Type string `json:"type,omitempty"`
	// ID is the identifier of the resource that has encountered a conflict.
	ID string `json:"id,omitempty"`
	// Reason is a natural language explanation of the conflict.
	Reason string `json:"reason,omitempty"`
}

func (e *ErrConflict) Error() string {
	return e.Reason
}

// ErrInternalServer represents a condition wherein the Bookkeeper server has
// encountered an unexpected error and does not wish to communicate further
// details of that error to the client.
type ErrInternalServer struct{}

func (e *ErrInternalServer) Error() string {
	return "An internal server error occurred."
}

// ErrNotSupported represents an error wherein a request cannot be completed
// because the Bookkeeper server explicitly does not support it. This can occur,
// for instance, if a client asks Bookkeeper to open a PR against an unsupported
// Git provider.
type ErrNotSupported struct {
	// Details is a natural language explanation of why the request was is not
	// supported by the Bookkeeper server.
	Details string `json:"reason,omitempty"`
}

func (e *ErrNotSupported) Error() string {
	return fmt.Sprintf("Request not supported: %s", e.Details)
}
