package api

import (
	"encoding/json"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

const (
	// MigratedProjectSpecToProjectConfig is the migration type for migrating
	// ProjectSpec resources to ProjectConfig resources. This migration is
	// performed by the controller when it detects that a Project resource
	// has a ProjectSpec resource in its spec, which is the case for Projects
	// created before the introduction of ProjectConfig resources.
	MigratedProjectSpecToProjectConfig = "ProjectSpecToProjectConfig"
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

// HasMigrationAnnotationValue checks if the AnnotationKeyMigrated annotation
// is present in the provided annotations map and if it contains the specified
// migration type as a key with a value of true.
func HasMigrationAnnotationValue(obj client.Object, migrationType string) bool {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		return false
	}
	migrated, ok := annotations[kargoapi.AnnotationKeyMigrated]
	if !ok {
		return false
	}
	var migrations map[string]bool
	if err := json.Unmarshal([]byte(migrated), &migrations); err != nil {
		return false
	}
	return migrations[migrationType]
}

// AddMigrationAnnotationValue updates the AnnotationKeyMigrated annotation
// in the provided annotations map to indicate that the specified migration
// type has been performed. If the annotation does not exist, it is created.
func AddMigrationAnnotationValue(obj client.Object, migrationType string) {
	annotations := obj.GetAnnotations()
	migrated, ok := annotations[kargoapi.AnnotationKeyMigrated]
	if !ok {
		migrated = "{}"
	}
	var migrations map[string]bool
	if err := json.Unmarshal([]byte(migrated), &migrations); err != nil {
		migrations = make(map[string]bool)
	}
	migrations[migrationType] = true
	b, _ := json.Marshal(migrations)
	if annotations == nil {
		annotations = make(map[string]string, 1)
	}
	annotations[kargoapi.AnnotationKeyMigrated] = string(b)
	obj.SetAnnotations(annotations)
}
