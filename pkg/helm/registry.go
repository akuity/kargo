package helm

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"

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
//
// When plainHTTP is true, the client communicates with the registry over plain
// HTTP instead of HTTPS.
func NewRegistryClient(authorizer auth.Client, plainHTTP bool) (*registry.Client, error) {
	opts := []registry.ClientOption{
		registry.ClientOptWriter(io.Discard),
		registry.ClientOptAuthorizer(authorizer),
		// NB: Options like ClientOptCache and ClientOptHTTPClient do not have
		// an effect on the registry client when using the authorizer, as they
		// are only set when NewClient constructs a new authorizer internally.
	}
	if plainHTTP {
		opts = append(opts, registry.ClientOptPlainHTTP())
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

	// insecure records whether TLS verification is skipped, so scheme probing
	// during Login uses the same TLS posture as the authorizer's real requests.
	insecure bool
}

// NewEphemeralAuthorizer creates a new EphemeralAuthorizer with an in-memory
// credentials store. This authorizer does not persist credentials to disk and
// is suitable for temporary operations where you do not want to store
// credentials permanently. When insecure is true, TLS certificate verification
// errors are ignored. This should be enabled only with great caution.
func NewEphemeralAuthorizer(insecure bool) *EphemeralAuthorizer {
	httpClient := &http.Client{
		Transport: retry.NewTransport(newRegistryTransport(insecure)),
	}

	store := credentials.NewMemoryStore()

	client := auth.Client{
		Client:     httpClient,
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(store),
	}
	client.SetUserAgent("Kargo/" + version.GetVersion().Version)

	return &EphemeralAuthorizer{
		Client:   client,
		Store:    store,
		insecure: insecure,
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
	reg.PlainHTTP = IsPlainHTTP(ctx, host, a.insecure)
	// NB: We set the authorizer on the registry client to ensure that the
	// HTTP transport is used by the login operation below.
	reg.Client = &a.Client

	return credentials.Login(ctx, a.Store, reg, auth.Credential{
		Username: username,
		Password: password,
	})
}

// newRegistryTransport returns an HTTP transport for talking to a registry.
// When insecure is true, TLS certificate verification is skipped.
func newRegistryTransport(insecure bool) *http.Transport {
	transport := cleanhttp.DefaultTransport()
	if insecure {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // #nosec G402 -- explicitly allowed by insecureSkipTLSVerify
		}
	}
	return transport
}

// IsPlainHTTP reports whether the OCI registry at host serves plain HTTP rather
// than HTTPS. It probes https://<host>/v2/ and treats evidence that the server
// spoke HTTP to an HTTPS request -- http.ErrSchemeMismatch, or a TLS
// record-header error whose first bytes spell "HTTP/" -- as conclusive.
// insecure must match the TLS posture of real registry traffic so the probe
// behaves identically.
//
// On an inconclusive result (for example, an unreachable host) it returns
// false, so callers default to HTTPS and surface the real error on the actual
// request.
//
// Detecting the scheme up front is necessary because oras-go, since v2.6.1,
// refuses to silently downgrade HTTPS to HTTP: it compares the scheme of the
// request it builds against the registry's responses, so the scheme must be
// chosen before the request is built.
func IsPlainHTTP(ctx context.Context, host string, insecure bool) bool {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://"+host+"/v2/",
		nil,
	)
	if err != nil {
		return false
	}

	client := &http.Client{Transport: newRegistryTransport(insecure)}
	// #nosec G704 -- host identifies the operator-configured registry the caller
	// is already about to contact; this probe only determines its scheme and
	// introduces no new SSRF surface.
	resp, err := client.Do(req)
	if err != nil {
		var tlsErr tls.RecordHeaderError
		return errors.Is(err, http.ErrSchemeMismatch) ||
			(errors.As(err, &tlsErr) && string(tlsErr.RecordHeader[:]) == "HTTP/")
	}
	_ = resp.Body.Close()
	return false
}
