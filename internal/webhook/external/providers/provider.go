package providers

import (
	"net/http"
)

type Provider interface {
	// GetRepository runs the providers authentication
	// mechanism against the request and then parses the
	// request body for the repository name for which the
	// event was generated.
	GetRepository(*http.Request) (string, error)
}
