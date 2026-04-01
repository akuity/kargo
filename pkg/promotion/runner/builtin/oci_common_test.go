package builtin

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/credentials"
)

func Test_parseOCIReference(t *testing.T) {
	tests := []struct {
		name       string
		imageRef   string
		assertions func(*testing.T, name.Reference, credentials.Type, error)
	}{
		{
			name:     "standard registry reference",
			imageRef: "registry.example.com/image:tag",
			assertions: func(t *testing.T, ref name.Reference, credType credentials.Type, err error) {
				require.NoError(t, err)
				require.NotNil(t, ref)
				assert.Equal(t, credentials.TypeImage, credType)
				assert.Equal(t, "registry.example.com/image:tag", ref.String())
			},
		},
		{
			name:     "OCI Helm reference",
			imageRef: "oci://registry.example.com/chart:1.0.0",
			assertions: func(t *testing.T, ref name.Reference, credType credentials.Type, err error) {
				require.NoError(t, err)
				require.NotNil(t, ref)
				assert.Equal(t, credentials.TypeHelm, credType)
				assert.Equal(t, "registry.example.com/chart:1.0.0", ref.String())
			},
		},
		{
			name:     "invalid reference",
			imageRef: "invalid::reference",
			assertions: func(t *testing.T, ref name.Reference, credType credentials.Type, err error) {
				assert.ErrorContains(t, err, "invalid image reference")
				assert.Nil(t, ref)
				assert.Empty(t, credType)
			},
		},
		{
			name:     "OCI reference with port",
			imageRef: "oci://localhost:5000/chart:latest",
			assertions: func(t *testing.T, ref name.Reference, credType credentials.Type, err error) {
				require.NoError(t, err)
				require.NotNil(t, ref)
				assert.Equal(t, credentials.TypeHelm, credType)
				assert.Equal(t, "localhost:5000/chart:latest", ref.String())
			},
		},
		{
			name:     "standard registry reference with port",
			imageRef: "an.internal.registry.com:5050/myrepo/myimage:latest",
			assertions: func(t *testing.T, ref name.Reference, credType credentials.Type, err error) {
				require.NoError(t, err)
				require.NotNil(t, ref)
				assert.Equal(t, credentials.TypeImage, credType)
				assert.Equal(t, "an.internal.registry.com:5050/myrepo/myimage:latest", ref.String())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, credType, err := parseOCIReference(tt.imageRef)
			tt.assertions(t, ref, credType, err)
		})
	}
}

func Test_buildOCIRemoteOptions(t *testing.T) {
	tests := []struct {
		name       string
		credsDB    credentials.Database
		imageRef   string
		assertions func(*testing.T, []remote.Option, error)
	}{
		{
			name:     "basic options without auth",
			credsDB:  &credentials.FakeDB{},
			imageRef: "registry.example.com/image:tag",
			assertions: func(t *testing.T, opts []remote.Option, err error) {
				require.NoError(t, err)
				assert.Len(t, opts, 2)
			},
		},
		{
			name: "options with authentication",
			credsDB: &credentials.FakeDB{
				GetFn: func(context.Context, string, credentials.Type, string) (*credentials.Credentials, error) {
					return &credentials.Credentials{
						Username: "user",
						Password: "pass",
					}, nil
				},
			},
			imageRef: "registry.example.com/image:tag",
			assertions: func(t *testing.T, opts []remote.Option, err error) {
				require.NoError(t, err)
				assert.Len(t, opts, 3)
			},
		},
		{
			name: "OCI Helm authentication",
			credsDB: &credentials.FakeDB{
				GetFn: func(
					_ context.Context,
					_ string,
					credType credentials.Type,
					repoURL string,
				) (*credentials.Credentials, error) {
					assert.Equal(t, credentials.TypeHelm, credType)
					assert.Equal(t, "oci://registry.example.com/chart", repoURL)
					return &credentials.Credentials{
						Username: "helm-user",
						Password: "helm-pass",
					}, nil
				},
			},
			imageRef: "oci://registry.example.com/chart:1.0.0",
			assertions: func(t *testing.T, opts []remote.Option, err error) {
				require.NoError(t, err)
				assert.Len(t, opts, 3)
			},
		},
		{
			name: "credentials error",
			credsDB: &credentials.FakeDB{
				GetFn: func(context.Context, string, credentials.Type, string) (*credentials.Credentials, error) {
					return nil, errors.New("credentials database error")
				},
			},
			imageRef: "registry.example.com/image:tag",
			assertions: func(t *testing.T, opts []remote.Option, err error) {
				assert.ErrorContains(t, err, "error obtaining credentials")
				assert.Nil(t, opts)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, credType, err := parseOCIReference(tt.imageRef)
			require.NoError(t, err)

			opts, err := buildOCIRemoteOptions(
				context.Background(), tt.credsDB, "fake-project",
				ref, credType, false,
			)
			tt.assertions(t, opts, err)
		})
	}
}

func Test_ociHTTPTransport(t *testing.T) {
	tests := []struct {
		name                  string
		insecureSkipTLSVerify bool
		assertions            func(*testing.T, *http.Transport)
	}{
		{
			name:                  "default TLS verification",
			insecureSkipTLSVerify: false,
			assertions: func(t *testing.T, transport *http.Transport) {
				require.NotNil(t, transport)
				if transport.TLSClientConfig != nil {
					assert.False(t, transport.TLSClientConfig.InsecureSkipVerify)
				}
			},
		},
		{
			name:                  "skip TLS verification",
			insecureSkipTLSVerify: true,
			assertions: func(t *testing.T, transport *http.Transport) {
				require.NotNil(t, transport)
				require.NotNil(t, transport.TLSClientConfig)
				assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := ociHTTPTransport(tt.insecureSkipTLSVerify)
			tt.assertions(t, transport)
		})
	}
}

func Test_ociAuthOption(t *testing.T) {
	tests := []struct {
		name       string
		credsDB    credentials.Database
		imageRef   string
		assertions func(*testing.T, remote.Option, error)
	}{
		{
			name:     "no credentials for image",
			credsDB:  &credentials.FakeDB{},
			imageRef: "registry.example.com/image:tag",
			assertions: func(t *testing.T, opt remote.Option, err error) {
				require.NoError(t, err)
				assert.Nil(t, opt)
			},
		},
		{
			name:     "no credentials for Helm",
			credsDB:  &credentials.FakeDB{},
			imageRef: "registry.example.com/chart:1.0.0",
			assertions: func(t *testing.T, opt remote.Option, err error) {
				require.NoError(t, err)
				assert.Nil(t, opt)
			},
		},
		{
			name: "valid image credentials",
			credsDB: &credentials.FakeDB{
				GetFn: func(
					_ context.Context,
					_ string,
					credType credentials.Type,
					repoURL string,
				) (*credentials.Credentials, error) {
					assert.Equal(t, credentials.TypeImage, credType)
					assert.Equal(t, "registry.example.com/image", repoURL)
					return &credentials.Credentials{
						Username: "user",
						Password: "pass",
					}, nil
				},
			},
			imageRef: "registry.example.com/image:tag",
			assertions: func(t *testing.T, opt remote.Option, err error) {
				require.NoError(t, err)
				require.NotNil(t, opt)
			},
		},
		{
			name: "valid Helm credentials with OCI prefix",
			credsDB: &credentials.FakeDB{
				GetFn: func(
					_ context.Context,
					_ string,
					credType credentials.Type,
					repoURL string,
				) (*credentials.Credentials, error) {
					assert.Equal(t, credentials.TypeHelm, credType)
					assert.Equal(t, "oci://registry.example.com/chart", repoURL)
					return &credentials.Credentials{
						Username: "helm-user",
						Password: "helm-pass",
					}, nil
				},
			},
			imageRef: "oci://registry.example.com/chart:1.0.0",
			assertions: func(t *testing.T, opt remote.Option, err error) {
				require.NoError(t, err)
				require.NotNil(t, opt)
			},
		},
		{
			name: "empty username and password",
			credsDB: &credentials.FakeDB{
				GetFn: func(context.Context, string, credentials.Type, string) (*credentials.Credentials, error) {
					return &credentials.Credentials{
						Username: "",
						Password: "",
					}, nil
				},
			},
			imageRef: "registry.example.com/image:tag",
			assertions: func(t *testing.T, opt remote.Option, err error) {
				require.NoError(t, err)
				assert.Nil(t, opt)
			},
		},
		{
			name: "credentials database error",
			credsDB: &credentials.FakeDB{
				GetFn: func(context.Context, string, credentials.Type, string) (*credentials.Credentials, error) {
					return nil, errors.New("credentials database error")
				},
			},
			imageRef: "registry.example.com/image:tag",
			assertions: func(t *testing.T, opt remote.Option, err error) {
				assert.ErrorContains(t, err, "error obtaining credentials")
				assert.ErrorContains(t, err, "credentials database error")
				assert.Nil(t, opt)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, credType, err := parseOCIReference(tt.imageRef)
			require.NoError(t, err)

			opt, err := ociAuthOption(
				context.Background(), tt.credsDB, "fake-project",
				ref, credType,
			)
			tt.assertions(t, opt, err)
		})
	}
}
