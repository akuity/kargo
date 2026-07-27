package server

import (
	"context"
	"encoding/json"
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

func Test_server_getClusterConfig(t *testing.T) {
	testConfig := &kargoapi.ClusterConfig{
		ObjectMeta: metav1.ObjectMeta{Name: api.ClusterConfigName},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/system/cluster-config",
		[]restTestCase{
			{
				name: "ClusterConfig does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "gets ClusterConfig",
				clientBuilder: fake.NewClientBuilder().WithObjects(testConfig),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ClusterConfig in the response
					config := &kargoapi.ClusterConfig{}
					err := json.Unmarshal(w.Body.Bytes(), config)
					require.NoError(t, err)
					require.Equal(t, api.ClusterConfigName, config.Name)
				},
			},
		},
	)
}

func Test_server_getClusterConfig_watch(t *testing.T) {
	testRESTWatchEndpoint(
		t, &config.ServerConfig{},
		"/v1beta1/system/cluster-config?watch=true",
		[]restWatchTestCase{
			{
				name:          "cluster config not found",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "watches cluster config successfully",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.ClusterConfig{
						ObjectMeta: metav1.ObjectMeta{Name: api.ClusterConfigName},
					},
				),
				operations: func(ctx context.Context, c client.Client) {
					// Update the cluster config to trigger a watch event
					// Fetch the current config first to get the resource version
					config := &kargoapi.ClusterConfig{}
					_ = c.Get(ctx, client.ObjectKey{Name: api.ClusterConfigName}, config)

					config.Spec.WebhookReceivers = []kargoapi.WebhookReceiverConfig{
						{Name: "new-receiver"},
					}
					_ = c.Update(ctx, config)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify SSE headers
					require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
					require.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
					require.Equal(t, "keep-alive", w.Header().Get("Connection"))

					// The response body should contain SSE events from the update operation
					body := w.Body.String()
					require.Contains(t, body, "data:")
				},
			},
		},
	)
}
