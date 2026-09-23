package event

import (
	corev1 "k8s.io/api/core/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// APIToken describes the API token an event is about. An API token is a
// ServiceAccount token Secret bound to a Kargo Role, so the Secret is the
// event's involved object and the Role is recorded alongside it.
type APIToken struct {
	Name     string `json:"name"`
	RoleName string `json:"roleName"`
}

func (a APIToken) GetName() string {
	return a.Name
}

func (a APIToken) Kind() string {
	return "Secret"
}

func (a *APIToken) MarshalAnnotationsTo(annotations map[string]string) {
	annotations[kargoapi.AnnotationKeyEventAPITokenName] = a.Name
	annotations[kargoapi.AnnotationKeyEventRoleName] = a.RoleName
}

// UnmarshalAPITokenAnnotations populates the APIToken fields from the given
// Kubernetes annotations.
func UnmarshalAPITokenAnnotations(annotations map[string]string) APIToken {
	return APIToken{
		Name:     annotations[kargoapi.AnnotationKeyEventAPITokenName],
		RoleName: annotations[kargoapi.AnnotationKeyEventRoleName],
	}
}

// APITokenCreated is emitted by the API server when it mints an API token.
type APITokenCreated struct {
	Common
	APIToken
}

func (a *APITokenCreated) Type() kargoapi.EventType {
	return kargoapi.EventTypeAPITokenCreated
}

// NewAPITokenCreated creates a new APITokenCreated event for the given token
// Secret. The Secret's namespace is the event's project; for a system-level
// token that is Kargo's own namespace.
func NewAPITokenCreated(message, actor string, tokenSecret *corev1.Secret) *APITokenCreated {
	evt := &APITokenCreated{
		Common: Common{Message: message},
	}
	if actor != "" {
		evt.Actor = &actor
	}
	if tokenSecret != nil {
		evt.Project = tokenSecret.Namespace
		evt.Name = tokenSecret.Name
		evt.RoleName = tokenSecret.Annotations[corev1.ServiceAccountNameKey]
	}
	return evt
}

func (a *APITokenCreated) MarshalAnnotations() map[string]string {
	annotations := map[string]string{}
	a.Common.MarshalAnnotationsTo(annotations)
	a.APIToken.MarshalAnnotationsTo(annotations)
	return annotations
}

// UnmarshalAPITokenCreatedAnnotations populates an APITokenCreated event from
// the given Kubernetes annotations and event ID.
func UnmarshalAPITokenCreatedAnnotations(
	eventID string,
	annotations map[string]string,
) (*APITokenCreated, error) {
	common, err := UnmarshalCommonAnnotations(eventID, annotations)
	if err != nil {
		return nil, err
	}
	return &APITokenCreated{
		Common:   common,
		APIToken: UnmarshalAPITokenAnnotations(annotations),
	}, nil
}
