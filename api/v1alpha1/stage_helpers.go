package v1alpha1

import (
	"encoding/json"
)

// VerificationRequest is a request payload with an optional actor field which
// can be used to annotate a Stage using the AnnotationKeyReverify or
// AnnotationKeyAbort annotations.
//
// The ID field is used to specify the VerificationInfo to be re-verified or
// aborted. If the ID is empty, the request is considered invalid. The Actor
// field is optional and can be used to track the user who initiated the
// re-verification or abort request. The ControlPlane field is optional and
// indicates if the request was initiated by a control plane.
//
// +protobuf=false
// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
type VerificationRequest struct {
	// ID is the identifier of the VerificationInfo for which the request is
	// being made.
	ID string `json:"id,omitempty"`
	// Actor is the user who initiated the request.
	Actor string `json:"actor,omitempty"`
	// ControlPlane is a flag to indicate if the request has been initiated by
	// a control plane.
	ControlPlane bool `json:"controlPlane,omitempty"`
}

// Equals returns true if the VerificationRequest is equal to the other
// VerificationRequest, false otherwise. Two VerificationRequests are equal if
// their ID, Actor, and ControlPlane fields are equal.
func (r *VerificationRequest) Equals(other *VerificationRequest) bool {
	if r == nil && other == nil {
		return true
	}
	if r == nil || other == nil {
		return false
	}
	return r.ID == other.ID && r.Actor == other.Actor && r.ControlPlane == other.ControlPlane
}

// HasID returns true if the VerificationRequest has a non-empty ID.
func (r *VerificationRequest) HasID() bool {
	return r != nil && r.ID != ""
}

// ForID returns true if the VerificationRequest has the specified ID.
func (r *VerificationRequest) ForID(id string) bool {
	return r != nil && r.ID != "" && r.ID == id
}

// String returns the JSON string representation of the VerificationRequest,
// or an empty string if the VerificationRequest is nil or has an empty ID.
func (r *VerificationRequest) String() string {
	if r == nil || r.ID == "" {
		return ""
	}
	b, _ := json.Marshal(r)
	if b == nil {
		return ""
	}
	return string(b)
}
