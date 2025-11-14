package helm

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"

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
			authorizer: NewEphemeralAuthorizer().Client,
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
	authorizer := NewEphemeralAuthorizer()

	assert.NotNil(t, authorizer)
	assert.NotNil(t, authorizer.Client)
	assert.NotNil(t, authorizer.Store)
	assert.NotNil(t, authorizer.Client.Client)
	assert.NotNil(t, authorizer.Cache)
	assert.NotNil(t, authorizer.Credential)

	// Verify the user agent is set correctly
	expectedUserAgent := "Kargo/" + version.GetVersion().Version
	assert.Contains(t, authorizer.Header.Get("User-Agent"), expectedUserAgent)
}
