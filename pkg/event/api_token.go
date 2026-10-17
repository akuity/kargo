package event

import (
	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	corev1 "k8s.io/api/core/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// APIToken describes the API token an event is about. An API token is a
// ServiceAccount token Secret bound to a Kargo Role, so the Secret is the
// object the event is about and the Role is recorded alongside it.
type APIToken struct {
	Name     string `json:"name"`
	RoleName string `json:"roleName"`
	// SystemLevel is true when the token belongs to a system-level Role. Such
	// tokens live in Kargo's own namespace, which the event otherwise reports
	// as its project.
	SystemLevel bool `json:"systemLevel"`
}

// APITokenCreated is emitted by the API server when it mints an API token.
type APITokenCreated struct {
	Common
	APIToken
}

// NewAPITokenCreated creates a new APITokenCreated event for the given token
// Secret. The Secret's namespace is the event's project; for a system-level
// token that is Kargo's own namespace.
func NewAPITokenCreated(
	message, actor string,
	tokenSecret *corev1.Secret,
	systemLevel bool,
) (cloudevents.Event, error) {
	common, token := newAPITokenParts(message, actor, tokenSecret, systemLevel)
	return newUserEvent(
		string(kargoapi.EventTypeAPITokenCreated),
		kindSecret,
		&APITokenCreated{Common: common, APIToken: token},
	)
}

// APITokenDeleted is emitted by the API server when it deletes an API token.
type APITokenDeleted struct {
	Common
	APIToken
}

// NewAPITokenDeleted creates a new APITokenDeleted event for the given token
// Secret, as it was before deletion.
func NewAPITokenDeleted(
	message, actor string,
	tokenSecret *corev1.Secret,
	systemLevel bool,
) (cloudevents.Event, error) {
	common, token := newAPITokenParts(message, actor, tokenSecret, systemLevel)
	return newUserEvent(
		string(kargoapi.EventTypeAPITokenDeleted),
		kindSecret,
		&APITokenDeleted{Common: common, APIToken: token},
	)
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
