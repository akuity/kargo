package dockerhub

import (
	"net/http"

	libHTTP "github.com/akuityio/k8sta/internal/common/http"
)

// TokenFilterConfig is the interface for a component that encapsulates token
// filter configuration.
type TokenFilterConfig interface {
	// AddToken adds a token to the TokenFilterConfig implementation's instance's
	// internal list of tokens.
	AddToken(string)
	// HasToken returns a bool indicating whether or not the TokenFilterConfig
	// implementation was configured to accept the provided token.
	HasToken(string) bool
}

// tokenFilterConfig encapsulates token filter configuration.
type tokenFilterConfig struct {
	tokens map[string]struct{}
}

// NewTokenFilterConfig returns an initialized implementation of the
// TokenFilterConfig interface.
func NewTokenFilterConfig() TokenFilterConfig {
	return &tokenFilterConfig{
		tokens: map[string]struct{}{},
	}
}

func (t *tokenFilterConfig) AddToken(token string) {
	t.tokens[token] = struct{}{}
}

func (t *tokenFilterConfig) HasToken(token string) bool {
	_, found := t.tokens[token]
	return found
}

// tokenFilter is a component that implements the http.Filter interface and can
// conditionally allow or disallow a request on the basis of a recognized token
// having been provided.
type tokenFilter struct {
	config TokenFilterConfig
}

// NewTokenFilter returns a component that implements the http.Filter interface
// and can conditionally allow or disallow a request on the basis of a
// recognized token having been provided.
func NewTokenFilter(config TokenFilterConfig) libHTTP.Filter {
	return &tokenFilter{
		config: config,
	}
}

func (t *tokenFilter) Decorate(handle http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Find the access token provided by the client.
		//
		// Note that Docker Hub doesn't support ANY reasonable webhook
		// authentication scheme AT ALL. The best we can possibly rely on is that
		// Docker Hub users can add a query parameter containing a token to the
		// URL they use when defining their webhooks. This is why we select the
		// token from a query parameter instead of a header.
		//
		// Further note that even with TLS in play, this is not *entirely* secure
		// because web servers, reverse proxies, and other infrastructure are apt
		// to capture entire URLs, including query parameters, in their access logs.
		//
		// User facing documentation does note these risks.
		providedToken := r.URL.Query().Get("access_token")
		// If no token was provided, then access is denied
		if providedToken == "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if !t.config.HasToken(providedToken) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		// If we get this far, everything checks out. Handle the request.
		handle(w, r)
	}
}
