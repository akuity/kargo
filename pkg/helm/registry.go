package helm

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"sync"

	"github.com/hashicorp/go-cleanhttp"
	"helm.sh/helm/v3/pkg/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"

	"github.com/akuity/kargo/pkg/x/version"
)

// NewRegistryClient creates a new registry client using the provided authorizer.
// This can be combined with an EphemeralAuthorizer to create a client that
// does not persist credentials to disk. The authorizer is used to authenticate
// requests to the registry, and the client is configured to write logs to
// io.Discard, meaning that it will not output any logs to the console or
// standard output.
func NewRegistryClient(authorizer auth.Client) (*registry.Client, error) {
	opts := []registry.ClientOption{
		registry.ClientOptWriter(io.Discard),
		registry.ClientOptAuthorizer(authorizer),
		// NB: Options like ClientOptCache and ClientOptHTTPClient do not have
		// an effect on the registry client when using the authorizer, as they
		// are only set when NewClient constructs a new authorizer internally.
	}
	return registry.NewClient(opts...)
}

// EphemeralAuthorizer provides a temporary authorizer for registry operations.
// It uses an in-memory credentials store and does not persist any credentials
// to disk. This is useful for ephemeral operations where you do not want to
// store credentials permanently. For example, because you are working with
// multiple tenants in a single process.
type EphemeralAuthorizer struct {
	auth.Client
	credentials.Store
}

// NewEphemeralAuthorizer creates a new EphemeralAuthorizer with an in-memory
// credentials store. This authorizer does not persist credentials to disk and
// is suitable for temporary operations where you do not want to store
// credentials permanently.
func NewEphemeralAuthorizer() *EphemeralAuthorizer {
	httpClient := cleanhttp.DefaultClient()
	httpClient.Transport = retry.NewTransport(newTransport(httpClient.Transport))

	store := credentials.NewMemoryStore()

	client := auth.Client{
		Client:     httpClient,
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(store),
	}
	client.SetUserAgent("Kargo/" + version.GetVersion().Version)

	return &EphemeralAuthorizer{
		Client: client,
		Store:  store,
	}
}

// Login logs in to the specified registry using the provided username and
// password. It uses the in-memory credentials store to save the credentials
// for the registry host. This method does not persist credentials to disk, so
// it is suitable for ephemeral operations where you do not want to store
// credentials permanently.
func (a *EphemeralAuthorizer) Login(ctx context.Context, host, username, password string) error {
	reg, err := remote.NewRegistry(host)
	if err != nil {
		return err
	}
	// NB: We set the authorizer on the registry client to ensure that the
	// HTTP transport is used by the login operation below.
	reg.Client = &a.Client

	return credentials.Login(ctx, a.Store, reg, auth.Credential{
		Username: username,
		Password: password,
	})
}

// fallbackTransport is a custom HTTP transport that allows for retrying
// requests that fail with a TLS RecordHeaderError by switching from HTTPS to
// HTTP.
//
// This was the default behavior for ORAS v1, but was removed in v2. We keep
// this transport to maintain compatibility with existing workflows that
// expect this behavior.
type fallbackTransport struct {
	Base      http.RoundTripper
	httpHosts sync.Map
}

// newTransport creates a new fallbackTransport with the provided base
// round tripper. It initializes the transport with a sync.Map to track hosts
// that should be retried with HTTP instead of HTTPS.
func newTransport(base http.RoundTripper) *fallbackTransport {
	return &fallbackTransport{
		Base: base,
	}
}

// RoundTrip wraps base round trip with conditional insecure retry.
func (t *fallbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	const (
		httpScheme  = "http"
		httpsScheme = "https"
	)

	host := req.URL.Host
	if forceHTTP, ok := t.httpHosts.Load(host); ok && forceHTTP.(bool) { // nolint:forcetypeassert
		req.URL.Scheme = httpScheme
		return t.Base.RoundTrip(req)
	}

	resp, err := t.Base.RoundTrip(req)
	if err != nil && req.URL.Scheme == httpsScheme {
		var tlsErr tls.RecordHeaderError
		if errors.As(err, &tlsErr) {
			if string(tlsErr.RecordHeader[:]) == "HTTP/" {
				t.httpHosts.Store(host, true)
				req.URL.Scheme = httpScheme
				return t.Base.RoundTrip(req)
			}
		}
	}
	return resp, err
}
