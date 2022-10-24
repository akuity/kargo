package server

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuityio/k8sta/internal/common/http"
	"github.com/akuityio/k8sta/internal/dockerhub"
)

func TestTokenFilterConfig(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func()
		assertions func(dockerhub.TokenFilterConfig, error)
	}{
		{
			name: "DOCKERHUB_TOKENS_PATH not set",
			assertions: func(_ dockerhub.TokenFilterConfig, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "value not found for")
				require.Contains(t, err.Error(), "DOCKERHUB_TOKENS_PATH")
			},
		},
		{
			name: "DOCKERHUB_TOKENS_PATH path does not exist",
			setup: func() {
				t.Setenv("DOCKERHUB_TOKENS_PATH", "/completely/bogus/path")
			},
			assertions: func(_ dockerhub.TokenFilterConfig, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"file /completely/bogus/path does not exist",
				)
			},
		},
		{
			name: "DOCKERHUB_TOKENS_PATH does not contain valid json",
			setup: func() {
				tokensFile, err := os.CreateTemp("", "tokens.json")
				require.NoError(t, err)
				defer tokensFile.Close()
				_, err = tokensFile.Write([]byte("this is not json"))
				require.NoError(t, err)
				t.Setenv("DOCKERHUB_TOKENS_PATH", tokensFile.Name())
			},
			assertions: func(_ dockerhub.TokenFilterConfig, err error) {
				require.Error(t, err)
				require.Contains(
					t, err.Error(), "invalid character",
				)
			},
		},
		{
			name: "success",
			setup: func() {
				tokensFile, err := os.CreateTemp("", "tokens.json")
				require.NoError(t, err)
				defer tokensFile.Close()
				_, err = tokensFile.Write([]byte(`{"foo": "bar"}`))
				require.NoError(t, err)
				t.Setenv("DOCKERHUB_TOKENS_PATH", tokensFile.Name())
			},
			assertions: func(config dockerhub.TokenFilterConfig, err error) {
				require.NoError(t, err)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			if testCase.setup != nil {
				testCase.setup()
			}
			config, err := dockerhubFilterConfig()
			testCase.assertions(config, err)
		})
	}
}

func TestServerConfig(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func()
		assertions func(http.ServerConfig, error)
	}{
		{
			name: "PORT not an int",
			setup: func() {
				t.Setenv("PORT", "foo")
			},
			assertions: func(_ http.ServerConfig, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "was not parsable as an int")
				require.Contains(t, err.Error(), "PORT")
			},
		},
		{
			name: "TLS_ENABLED not a bool",
			setup: func() {
				t.Setenv("PORT", "8080")
				t.Setenv("TLS_ENABLED", "nope")
			},
			assertions: func(_ http.ServerConfig, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "was not parsable as a bool")
				require.Contains(t, err.Error(), "TLS_ENABLED")
			},
		},
		{
			name: "TLS_CERT_PATH required but not set",
			setup: func() {
				t.Setenv("TLS_ENABLED", "true")
			},
			assertions: func(_ http.ServerConfig, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "value not found for")
				require.Contains(t, err.Error(), "TLS_CERT_PATH")
			},
		},
		{
			name: "TLS_KEY_PATH required but not set",
			setup: func() {
				t.Setenv("TLS_CERT_PATH", "/var/ssl/cert")
			},
			assertions: func(_ http.ServerConfig, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "value not found for")
				require.Contains(t, err.Error(), "TLS_KEY_PATH")
			},
		},
		{
			name: "success",
			setup: func() {
				t.Setenv("TLS_KEY_PATH", "/var/ssl/key")
			},
			assertions: func(config http.ServerConfig, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					http.ServerConfig{
						Port:        8080,
						TLSEnabled:  true,
						TLSCertPath: "/var/ssl/cert",
						TLSKeyPath:  "/var/ssl/key",
					},
					config,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.setup()
			config, err := serverConfig()
			testCase.assertions(config, err)
		})
	}
}
