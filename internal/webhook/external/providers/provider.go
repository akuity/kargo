package providers

import (
	"errors"
	"net/http"

	"github.com/akuity/kargo/internal/webhook/external/events"
	"github.com/akuity/kargo/internal/webhook/external/providers/github"
)

// ErrUnsupportedProvider returns when name of the provider passed
// is one no implementation currently exists for.
var ErrUnsupportedProvider = errors.New("unsupported provider")

type Provider interface {
	// Authenticate runs the providers authentication
	// mechanism against the request.
	Authenticate(*http.Request) error
	// Event returns a generic read-only event from the request
	Event(*http.Request) (events.Event, error)
}

func New(name Name) (Provider, error) {
	switch name {
	case Github:
		return github.NewProvider()
	// TODO(Faris): Support additional providers
	default:
		return nil, ErrUnsupportedProvider
	}
}

type Name int

func (name Name) String() string {
	switch name {
	case Github:
		return "github"
	// TODO(Faris): Support additional providers
	default:
		return "unknown"
	}
}

const (
	// Github is the name of the github provider
	Github Name = iota
	// TODO(Faris): Support additional providers
)
