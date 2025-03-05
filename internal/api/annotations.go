package api

import (
	"encoding/json"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// RefreshAnnotationValue returns the value of the AnnotationKeyRefresh
// annotation which can be used to detect changes, and a boolean indicating
// whether the annotation was present.
func RefreshAnnotationValue(annotations map[string]string) (string, bool) {
	requested, ok := annotations[kargoapi.AnnotationKeyRefresh]
	return requested, ok
}

// ReverifyAnnotationValue returns the value of the AnnotationKeyReverify
// annotation, which can be used to determine whether the verification of a
// Freight should be rerun, and a boolean indicating whether the annotation was
// present.
//
// If the value of the annotation is a valid JSON object, it is unmarshalled
// into a VerificationRequest struct. Otherwise, the value is treated as the ID
// of the verification to be reverified and set as the ID field of the returned
// VerificationRequest.
func ReverifyAnnotationValue(annotations map[string]string) (*kargoapi.VerificationRequest, bool) {
	requested, ok := annotations[kargoapi.AnnotationKeyReverify]
	if !ok {
		return nil, ok
	}
	var req kargoapi.VerificationRequest
	if b := []byte(requested); json.Valid(b) {
		if err := json.Unmarshal(b, &req); err != nil {
			return nil, false
		}
	} else {
		req.ID = requested
	}
	if !req.HasID() {
		return nil, false
	}
	return &req, ok
}

// AbortVerificationAnnotationValue returns the value of the AnnotationKeyAbort annotation
// which can be used to abort the verification of a Freight, and a boolean
// indicating whether the annotation was present.
//
// If the value of the annotation is a valid JSON object, it is unmarshalled
// into a VerificationRequest struct. Otherwise, the value is treated as the ID
// of the verification to be aborted and set as the ID field of the returned
// VerificationRequest.
func AbortVerificationAnnotationValue(annotations map[string]string) (*kargoapi.VerificationRequest, bool) {
	requested, ok := annotations[kargoapi.AnnotationKeyAbort]
	if !ok {
		return nil, ok
	}
	var req kargoapi.VerificationRequest
	if b := []byte(requested); json.Valid(b) {
		if err := json.Unmarshal(b, &req); err != nil {
			return nil, false
		}
	} else {
		req.ID = requested
	}
	if !req.HasID() {
		return nil, false
	}
	return &req, ok
}

// AbortPromotionAnnotationValue returns the value of the AnnotationKeyAbort
// annotation which can be used to abort the promotion of a Freight, and a
// boolean indicating whether the annotation was present.
//
// If the value of the annotation is a valid JSON object, it is unmarshalled
// into an AbortPromotionRequest struct. Otherwise, the value is treated as the
// action to be taken on the Promotion and set as the Action field of the
// returned AbortPromotionRequest.
func AbortPromotionAnnotationValue(annotations map[string]string) (*kargoapi.AbortPromotionRequest, bool) {
	requested, ok := annotations[kargoapi.AnnotationKeyAbort]
	if !ok {
		return nil, ok
	}
	var req kargoapi.AbortPromotionRequest
	if b := []byte(requested); json.Valid(b) {
		if err := json.Unmarshal(b, &req); err != nil {
			return nil, false
		}
	} else {
		req.Action = kargoapi.AbortAction(requested)
	}
	if req.Action == "" {
		return nil, false
	}
	return &req, ok
}
