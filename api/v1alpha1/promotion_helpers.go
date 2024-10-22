package v1alpha1

import (
	"encoding/json"
)

// AbortAction is an action to take on a Promotion to abort it.
type AbortAction string

const (
	// AbortActionTerminate is an action to terminate the Promotion.
	// I.e. the Promotion will be marked as failed and the controller
	// will stop processing it.
	AbortActionTerminate AbortAction = "terminate"
)

// AbortPromotionRequest is a request payload with an optional actor field which
// can be used to annotate a Promotion using the AnnotationKeyAbort annotation.
//
// +protobuf=false
// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
type AbortPromotionRequest struct {
	// Action is the action to take on the Promotion to abort it.
	Action AbortAction `json:"action,omitempty" protobuf:"bytes,1,opt,name=action"`
	// Actor is the user who initiated the request.
	Actor string `json:"actor,omitempty" protobuf:"bytes,2,opt,name=actor"`
	// ControlPlane is a flag to indicate if the request has been initiated by
	// a control plane.
	ControlPlane bool `json:"controlPlane,omitempty" protobuf:"varint,3,opt,name=controlPlane"`
}

// Equals returns true if the AbortPromotionRequest is equal to the other
// AbortPromotionRequest, false otherwise. Two VerificationRequests are equal
// if their Action, Actor, and ControlPlane fields are equal.
func (r *AbortPromotionRequest) Equals(other *AbortPromotionRequest) bool {
	if r == nil && other == nil {
		return true
	}
	if r == nil || other == nil {
		return false
	}
	return r.Action == other.Action && r.Actor == other.Actor && r.ControlPlane == other.ControlPlane
}

// String returns the JSON string representation of the AbortPromotionRequest,
// or an empty string if the AbortPromotionRequest is nil or has an empty Action.
func (r *AbortPromotionRequest) String() string {
	if r == nil || r.Action == "" {
		return ""
	}
	b, _ := json.Marshal(r)
	if b == nil {
		return ""
	}
	return string(b)
}
