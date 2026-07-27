package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_getConfig(t *testing.T) {
	testRESTEndpoint(
		t, &config.ServerConfig{
			SecretManagementEnabled: true,
			ArgoCDConfig: config.ArgoCDConfig{
				URLs: map[string]string{
					"": "https://argocd.example.com",
				},
			},
		},
		http.MethodGet, "/v1beta1/system/server-config",
		[]restTestCase{
			{
				name: "gets system config",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the response
					res := &getConfigResponse{}
					err := json.Unmarshal(w.Body.Bytes(), res)
					require.NoError(t, err)
					require.True(t, res.SecretManagementEnabled)
				},
			},
		},
	)
}
