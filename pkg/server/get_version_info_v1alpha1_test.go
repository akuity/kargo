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
	"github.com/akuity/kargo/pkg/x/edition"
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
					var response struct {
						version.Version
						Edition edition.Edition `json:"edition"`
					}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
					require.NotEmpty(t, response.Version)
					require.Equal(t, edition.Community, response.Edition)
				},
			},
		},
	)
}

func Test_server_getVersionInfo_edition(t *testing.T) {
	testRESTEndpoint(
		t,
		&config.ServerConfig{Edition: edition.Enterprise},
		http.MethodGet,
		"/v1beta1/system/server-version",
		[]restTestCase{{
			name: "gets enterprise edition",
			assertions: func(
				t *testing.T,
				w *httptest.ResponseRecorder,
				_ client.Client,
			) {
				require.Equal(t, http.StatusOK, w.Code)
				var response struct {
					Edition edition.Edition `json:"edition"`
				}
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
				require.Equal(t, edition.Enterprise, response.Edition)
			},
		}},
	)
}

func Test_server_GetVersionInfo_edition(t *testing.T) {
	s := &server{
		cfg: config.ServerConfig{Edition: edition.Enterprise},
	}

	response, err := s.GetVersionInfo(
		t.Context(),
		connect.NewRequest(&svcv1alpha1.GetVersionInfoRequest{}),
	)

	require.NoError(t, err)
	require.Equal(
		t,
		svcv1alpha1.ProductEdition_PRODUCT_EDITION_ENTERPRISE,
		response.Msg.VersionInfo.Edition,
	)
}
