package server

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuityio/kargo/internal/common/http"
)

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
