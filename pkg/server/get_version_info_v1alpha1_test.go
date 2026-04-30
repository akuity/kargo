package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/x/version"
)

func Test_server_getVersionInfo(t *testing.T) {
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/system/server-version", []restTestCase{
			{
				name: "gets version info",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					v := version.Version{}
					err := json.Unmarshal(w.Body.Bytes(), &v)
					require.NoError(t, err)
					require.NotEmpty(t, v.Version)
				},
			},
		},
	)
}
