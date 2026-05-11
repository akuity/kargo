package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/oidc"
)

func Test_server_getPublicConfig(t *testing.T) {
	testConfig := config.ServerConfig{
		AdminConfig: &config.AdminConfig{},
		OIDCConfig: &oidc.Config{
			IssuerURL:        "https://issuer.example.com",
			ClientID:         "client-id",
			CLIClientID:      "cli-client-id",
			DefaultScopes:    []string{"openid", "profile"},
			AdditionalScopes: []string{"email"},
		},
	}
	testRESTEndpoint(
		t, &testConfig,
		http.MethodGet, "/v1beta1/system/public-server-config",
		[]restTestCase{
			{
				name: "get public config with OIDC",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					var config publicConfig
					err := json.Unmarshal(w.Body.Bytes(), &config)
					require.NoError(t, err)
					require.True(t, config.AdminAccountEnabled)
					require.NotNil(t, config.OIDCConfig)
					require.Equal(t, testConfig.OIDCConfig.IssuerURL, config.OIDCConfig.IssuerURL)
					require.Equal(t, testConfig.OIDCConfig.ClientID, config.OIDCConfig.ClientID)
					require.Equal(t, testConfig.OIDCConfig.CLIClientID, config.OIDCConfig.CLIClientID)
					require.Equal(t, []string{"openid", "profile", "email"}, config.OIDCConfig.Scopes)
				},
			},
		},
	)
}
