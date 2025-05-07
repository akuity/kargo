package github

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	gh "github.com/google/go-github/github"
)

var (
	// ErrMissingSignature is returned when the 'X-Hub-Signature-256'
	// header is not found or empty.
	ErrMissingSignature = errors.New("missing signature")
	// ErrInvalidSignature is returned when the 'X-Hub-Signature-256'
	// header is not the value that was expected.
	ErrInvalidSignature = errors.New("invalid signature")
	// ErrSecretUnset is returned when the 'GH_WEBHOOK_SECRET'
	// environment variable is empty.
	ErrSecretUnset = errors.New("secret is unset")
	// ErrUnsupportedEventType is returned when the received
	// request body does not conform to a github push event.
	ErrUnsupportedEventType = errors.New("unsupported event type")
)

type provider struct {
	secret  string
	payload []byte
}

func NewProvider() (*provider, error) {
	secret, ok := os.LookupEnv("GH_WEBHOOK_SECRET")
	if !ok {
		return nil, ErrSecretUnset
	}
	return &provider{secret: secret}, nil
}

func (p *provider) Name() string {
	return "github"
}

func (p *provider) Authenticate(r *http.Request) error {
	payload, err := gh.ValidatePayload(r, []byte(p.secret))
	if err != nil {
		return fmt.Errorf("failed to validate payload: %w", err)
	}
	p.payload = payload
	return nil
}

func (p *provider) Repository(r *http.Request) (string, error) {
	event, err := gh.ParseWebHook(gh.WebHookType(r), p.payload)
	if err != nil {
		return "", fmt.Errorf("failed to parse webhook event: %w", err)
	}
	pe, ok := event.(*gh.PushEvent)
	if !ok {
		return "", ErrUnsupportedEventType
	}
	return *pe.Repo.HTMLURL, nil
}
