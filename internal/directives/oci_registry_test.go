package directives

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/google/uuid"
)

// authRegistryHandler is a http.Handler that wraps another http.Handler and
// requires clients to exchange a username and password for a Bearer token to
// access the underlying handler.
type authRegistryHandler struct {
	// registry is the underlying registry handler that this handler wraps.
	registry http.Handler

	// username and password are the credentials that clients must provide to
	// receive a Bearer token.
	username string
	password string

	// token is the Bearer token that clients must provide to access the
	// registry. It is randomly generated when the handler is created.
	token string
}

// newAuthRegistryServer creates a new httptest.Server that wraps a registry
// handler and requires clients to exchange a username and password for a Bearer
// token to access the underlying handler.
func newAuthRegistryServer(username, password string, opts ...registry.Option) *httptest.Server {
	return httptest.NewUnstartedServer(
		newAuthRegistryHandler(
			registry.New(opts...),
			username,
			password,
		),
	)
}

// newAuthRegistryHandler creates a new authRegistryHandler that wraps the given
// registry handler and requires clients to exchange a username and password for
// a Bearer token to access the underlying handler.
func newAuthRegistryHandler(registryHandler http.Handler, username, password string) http.Handler {
	token := base64.StdEncoding.EncodeToString([]byte(uuid.New().String()))

	return &authRegistryHandler{
		registry: registryHandler,
		username: username,
		password: password,
		token:    token,
	}
}

// ServeHTTP implements the http.Handler interface.
func (m *authRegistryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	const authEndpoint = "/authenticate"

	// If the request is for the authentication endpoint, check the provided
	// credentials and return a Bearer token if they are correct.
	if r.URL.Path == authEndpoint {
		username, password, ok := r.BasicAuth()
		if !ok || username != m.username || password != m.password {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"token": "%s"}`, m.token)))
		return
	}

	// If the request is not for the authentication endpoint, check that the
	// client has provided a valid Bearer token.
	if r.Header.Get("Authorization") == fmt.Sprintf("Bearer %s", m.token) {
		m.registry.ServeHTTP(w, r)
		return
	}

	// If the client has not provided a valid Bearer token, return a 401
	// Unauthorized response with a WWW-Authenticate header that tells the client
	// how to authenticate.
	realm := m.host(r) + authEndpoint
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Www-Authenticate", fmt.Sprintf("Bearer realm=%q", realm))
	w.WriteHeader(http.StatusUnauthorized)
}

// host returns the base URL of the server that is handling the request.
func (m *authRegistryHandler) host(r *http.Request) string {
	protocol := "http"
	if r.TLS != nil {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s", protocol, r.Host)
}
