package helm

import (
	"net/http"
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
			client, err := NewRegistryClient(tt.authorizer)
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

		// Unwrap the transport chain: retry.Transport -> fallbackTransport ->
		// *http.Transport to verify InsecureSkipVerify is set.
		retryT, ok := authorizer.Client.Client.Transport.(*retry.Transport)
		assert.True(t, ok, "expected outermost transport to be *retry.Transport")
		if ok {
			fallbackT, ok := retryT.Base.(*fallbackTransport)
			assert.True(t, ok, "expected inner transport to be *fallbackTransport")
			if ok {
				httpT, ok := fallbackT.Base.(*http.Transport)
				assert.True(t, ok, "expected base transport to be *http.Transport")
				if ok {
					assert.NotNil(t, httpT.TLSClientConfig)
					assert.True(t, httpT.TLSClientConfig.InsecureSkipVerify)
				}
			}
		}
	})
}
