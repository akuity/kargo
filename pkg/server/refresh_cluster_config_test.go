package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_refreshClusterConfig(t *testing.T) {
	testConfig := &kargoapi.ClusterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: api.ClusterConfigName},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost, "/v1beta1/system/cluster-config/refresh",
		[]restTestCase{
			{
				name:          "ClusterConfig not found",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "refreshes ClusterConfig",
				clientBuilder: fake.NewClientBuilder().WithObjects(testConfig),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify the ClusterConfig was refreshed
					config := &kargoapi.ClusterConfig{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testConfig),
						config,
					)
					require.NoError(t, err)
					require.NotEmpty(t, config.Annotations[kargoapi.AnnotationKeyRefresh])
				},
			},
		},
	)
}
