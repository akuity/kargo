package providers

import (
	"net/http"
)

type Provider interface {
	// Authenticate runs the providers authentication
	// mechanism against the request.
	Authenticate(*http.Request) error
	// Repository returns the repository name for which the event was generated.
	Repository(*http.Request) (string, error)
}
