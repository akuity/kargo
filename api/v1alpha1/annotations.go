package v1alpha1

import "encoding/json"

const (
	// AnnotationKeyAzureClientID is an annotation key that can be set on a
	// Project resource to indicate the client ID of an Azure managed identity
	// that the Project is indirectly associated with.
	AnnotationKeyAzureClientID = "azure.kargo.akuity.io/client-id"

	// AnnotationKeyCreateActor is an annotation key that can be injected to a
	// resource by the Kargo control plane to indicate the actor that created
	// the resource.
	AnnotationKeyCreateActor = "kargo.akuity.io/create-actor"

	// AnnotationKeyRefresh is an annotation key that can be set on a resource
	// to trigger a refresh of the resource by the controller. The value of the
	// annotation is interpreted as a token, and any change to the value of the
	// annotation should trigger a reconciliation of the resource.
	AnnotationKeyRefresh = "kargo.akuity.io/refresh"

	// AnnotationKeyReverify is an annotation key that can be set on a Stage
	// resource to trigger the re-verification of its Freight. The value of the
	// annotation should either be the ID of the verification to be reverified,
	// or a JSON object with the structure of the VerificationRequest.
	AnnotationKeyReverify = "kargo.akuity.io/reverify"

	// AnnotationKeyAbort is an annotation key that can be set on a Stage
	// resource to abort the verification of its Freight. The value of the
	// annotation must be set to the identifier of the verification to be
	// aborted.
	AnnotationKeyAbort = "kargo.akuity.io/abort"

	// AnnotationKeyDescription is an annotation key that can be set on a
	// resource to provide a description of it. The value of the annotation may
	// be used by the Kargo UI to display additional information about the
	// resource.
	AnnotationKeyDescription = "kargo.akuity.io/description"

	AnnotationValueTrue = "true"
)

// RefreshAnnotationValue returns the value of the AnnotationKeyRefresh
// annotation which can be used to detect changes, and a boolean indicating
// whether the annotation was present.
func RefreshAnnotationValue(annotations map[string]string) (string, bool) {
	requested, ok := annotations[AnnotationKeyRefresh]
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
func ReverifyAnnotationValue(annotations map[string]string) (*VerificationRequest, bool) {
	requested, ok := annotations[AnnotationKeyReverify]
	if !ok {
		return nil, ok
	}
	var vr VerificationRequest
	if b := []byte(requested); json.Valid(b) {
		if err := json.Unmarshal(b, &vr); err != nil {
			return nil, false
		}
	} else {
		vr.ID = requested
	}
	if !vr.HasID() {
		return nil, false
	}
	return &vr, ok
}

// AbortAnnotationValue returns the value of the AnnotationKeyAbort annotation
// which can be used to abort the verification of a Freight, and a boolean
// indicating whether the annotation was present.
//
// If the value of the annotation is a valid JSON object, it is unmarshalled
// into a VerificationRequest struct. Otherwise, the value is treated as the ID
// of the verification to be aborted and set as the ID field of the returned
// VerificationRequest.
func AbortAnnotationValue(annotations map[string]string) (*VerificationRequest, bool) {
	requested, ok := annotations[AnnotationKeyAbort]
	if !ok {
		return nil, ok
	}
	var vr VerificationRequest
	if b := []byte(requested); json.Valid(b) {
		if err := json.Unmarshal(b, &vr); err != nil {
			return nil, false
		}
	} else {
		vr.ID = requested
	}
	if !vr.HasID() {
		return nil, false
	}
	return &vr, ok
}
