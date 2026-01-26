package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func TestGetConfig(t *testing.T) {
	testCases := map[string]struct {
		req        *svcv1alpha1.GetConfigRequest
		cfg        config.ServerConfig
		assertions func(res *svcv1alpha1.GetConfigResponse)
	}{
		"get config": {
			req: &svcv1alpha1.GetConfigRequest{},
			cfg: config.ServerConfig{
				ArgoCDConfig: config.ArgoCDConfig{
					URLs: map[string]string{
						"": "https://argocd.example.com",
					},
				},
			},
			assertions: func(res *svcv1alpha1.GetConfigResponse) {
				require.Equal(t, "argocd", res.ArgocdShards[""].Namespace)
				require.Equal(t, "https://argocd.example.com", res.ArgocdShards[""].Url)
			},
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			svr := &server{
				cfg: tc.cfg,
			}
			res, err := svr.GetConfig(t.Context(), connect.NewRequest(tc.req))
			require.NoError(t, err)
			if tc.assertions != nil {
				tc.assertions(res.Msg)
			}
		})
	}
}

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
