package helm

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"

	"github.com/akuity/kargo/pkg/x/version"
)

func TestNewRegistryClient(t *testing.T) {
	tests := []struct {
		name       string
		authorizer auth.Client
		assertions func(t *testing.T, client any, err error)
	}{
		{
			name:       "successful creation with ephemeral authorizer",
			authorizer: NewEphemeralAuthorizer(false).Client,
			assertions: func(t *testing.T, client any, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			},
		},
		{
			name: "successful creation with custom authorizer",
			authorizer: auth.Client{
				Client:     http.DefaultClient,
				Cache:      auth.NewCache(),
				Credential: credentials.Credential(credentials.NewMemoryStore()),
			},
			assertions: func(t *testing.T, client any, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, client)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewRegistryClient(tt.authorizer, false)
			tt.assertions(t, client, err)
		})
	}
}

func TestNewEphemeralAuthorizer(t *testing.T) {
	t.Run("skipping tls cert verification", func(t *testing.T) {
		authorizer := NewEphemeralAuthorizer(false)

		assert.NotNil(t, authorizer)
		assert.NotNil(t, authorizer.Client)
		assert.NotNil(t, authorizer.Store)
		assert.NotNil(t, authorizer.Client.Client)
		assert.NotNil(t, authorizer.Cache)
		assert.NotNil(t, authorizer.Credential)

		// Verify the user agent is set correctly
		expectedUserAgent := "Kargo/" + version.GetVersion().Version
		assert.Contains(t, authorizer.Header.Get("User-Agent"), expectedUserAgent)
	})

	t.Run("insecure", func(t *testing.T) {
		authorizer := NewEphemeralAuthorizer(true)

		assert.NotNil(t, authorizer)
		assert.NotNil(t, authorizer.Client.Client)

		// Unwrap the transport chain: retry.Transport -> *http.Transport to
		// verify InsecureSkipVerify is set.
		retryT, retryOK := authorizer.Client.Client.Transport.(*retry.Transport)
		assert.True(t, retryOK, "expected outermost transport to be *retry.Transport")
		if retryOK {
			httpT, httpOK := retryT.Base.(*http.Transport)
			assert.True(t, httpOK, "expected base transport to be *http.Transport")
			if httpOK {
				assert.NotNil(t, httpT.TLSClientConfig)
				assert.True(t, httpT.TLSClientConfig.InsecureSkipVerify)
			}
		}

		assert.True(t, authorizer.insecure)
	})
}

func TestIsPlainHTTP(t *testing.T) {
	t.Run("plain HTTP registry is detected", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		))
		t.Cleanup(server.Close)

		assert.True(t, IsPlainHTTP(t.Context(), hostOf(t, server.URL), false))
	})

	t.Run("HTTPS registry is not plain HTTP", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(
			func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		))
		t.Cleanup(server.Close)

		// insecure is required to accept the server's self-signed certificate.
		assert.False(t, IsPlainHTTP(t.Context(), hostOf(t, server.URL), true))
	})

	t.Run("unreachable host defaults to HTTPS", func(t *testing.T) {
		// Reserve a port and immediately release it so nothing is listening.
		server := httptest.NewServer(http.HandlerFunc(
			func(http.ResponseWriter, *http.Request) {},
		))
		host := hostOf(t, server.URL)
		server.Close()

		assert.False(t, IsPlainHTTP(t.Context(), host, false))
	})
}

func hostOf(t *testing.T, serverURL string) string {
	t.Helper()
	return strings.TrimPrefix(strings.TrimPrefix(serverURL, "https://"), "http://")
}
