package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
)

func TestSetupRESTRouter_authentication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name   string
		cfg    config.ServerConfig
		path   string
		assert func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "admin and OIDC disabled; protected path without a token",
			path: "/v1beta1/system/server-config",
			assert: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, w.Code)
			},
		},
		{
			name: "admin and OIDC disabled; exempt path without a token",
			path: "/v1beta1/system/public-server-config",
			assert: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
			},
		},
		{
			name: "local mode; protected path without a token",
			cfg:  config.ServerConfig{LocalMode: true},
			path: "/v1beta1/system/server-config",
			assert: func(t *testing.T, w *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, w.Code)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			s := &server{cfg: testCase.cfg}
			var err error
			s.client, err = kubernetes.NewClient(
				t.Context(),
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						context.Context,
						*rest.Config,
						*runtime.Scheme,
						string,
					) (client.WithWatch, error) {
						return fake.NewClientBuilder().Build(), nil
					},
				},
			)
			require.NoError(t, err)
			router := s.setupRESTRouter(t.Context())
			w := httptest.NewRecorder()
			router.ServeHTTP(
				w,
				httptest.NewRequest(http.MethodGet, testCase.path, nil),
			)
			testCase.assert(t, w)
		})
	}
}
