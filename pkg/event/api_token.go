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
	// SystemLevel is true when the token belongs to a system-level Role. Such
	// tokens live in Kargo's own namespace, which the event otherwise reports
	// as its project.
	SystemLevel bool `json:"systemLevel"`
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
	if a.SystemLevel {
		annotations[kargoapi.AnnotationKeyEventAPITokenSystemLevel] = kargoapi.AnnotationValueTrue
	}
}

// UnmarshalAPITokenAnnotations populates the APIToken fields from the given
// Kubernetes annotations.
func UnmarshalAPITokenAnnotations(annotations map[string]string) APIToken {
	return APIToken{
		Name:        annotations[kargoapi.AnnotationKeyEventAPITokenName],
		RoleName:    annotations[kargoapi.AnnotationKeyEventRoleName],
		SystemLevel: annotations[kargoapi.AnnotationKeyEventAPITokenSystemLevel] == kargoapi.AnnotationValueTrue,
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
func NewAPITokenCreated(
	message, actor string,
	tokenSecret *corev1.Secret,
	systemLevel bool,
) *APITokenCreated {
	common, token := newAPITokenParts(message, actor, tokenSecret, systemLevel)
	return &APITokenCreated{Common: common, APIToken: token}
}

func (a *APITokenCreated) MarshalAnnotations() map[string]string {
	return marshalAPITokenAnnotations(&a.Common, &a.APIToken)
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

// APITokenDeleted is emitted by the API server when it deletes an API token.
type APITokenDeleted struct {
	Common
	APIToken
}

func (a *APITokenDeleted) Type() kargoapi.EventType {
	return kargoapi.EventTypeAPITokenDeleted
}

// NewAPITokenDeleted creates a new APITokenDeleted event for the given token
// Secret, as it was before deletion.
func NewAPITokenDeleted(
	message, actor string,
	tokenSecret *corev1.Secret,
	systemLevel bool,
) *APITokenDeleted {
	common, token := newAPITokenParts(message, actor, tokenSecret, systemLevel)
	return &APITokenDeleted{Common: common, APIToken: token}
}

func (a *APITokenDeleted) MarshalAnnotations() map[string]string {
	return marshalAPITokenAnnotations(&a.Common, &a.APIToken)
}

// UnmarshalAPITokenDeletedAnnotations populates an APITokenDeleted event from
// the given Kubernetes annotations and event ID.
func UnmarshalAPITokenDeletedAnnotations(
	eventID string,
	annotations map[string]string,
) (*APITokenDeleted, error) {
	common, err := UnmarshalCommonAnnotations(eventID, annotations)
	if err != nil {
		return nil, err
	}
	return &APITokenDeleted{
		Common:   common,
		APIToken: UnmarshalAPITokenAnnotations(annotations),
	}, nil
}

func newAPITokenParts(
	message, actor string,
	tokenSecret *corev1.Secret,
	systemLevel bool,
) (Common, APIToken) {
	common := Common{Message: message}
	if actor != "" {
		common.Actor = &actor
	}
	token := APIToken{SystemLevel: systemLevel}
	if tokenSecret != nil {
		common.Project = tokenSecret.Namespace
		token.Name = tokenSecret.Name
		token.RoleName = tokenSecret.Annotations[corev1.ServiceAccountNameKey]
	}
	return common, token
}

func marshalAPITokenAnnotations(common *Common, token *APIToken) map[string]string {
	annotations := map[string]string{}
	common.MarshalAnnotationsTo(annotations)
	token.MarshalAnnotationsTo(annotations)
	return annotations
}
