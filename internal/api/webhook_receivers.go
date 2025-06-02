package api

// WebhookReceiverType is an enum representing the type of a webhook receiver.
// It is used to identify the platform or service that the webhook receiver is
// associated with, such as GitHub or Quay.
type WebhookReceiverType string

func (w WebhookReceiverType) String() string {
	return string(w)
}

const (
	WebhookReceiverTypeGitHub WebhookReceiverType = "GitHub"
	WebhookReceiverTypeQuay   WebhookReceiverType = "Quay"
	// TODO(fuskovic): Add more receiver enum types(e.g. Dockerhub, Quay, Gitlab, etc...)
)

const (
	WebhookReceiverSecretKeyGithub string = "token"
	WebhookReceiverSecretKeyQuay   string = "quay-secret"
)
