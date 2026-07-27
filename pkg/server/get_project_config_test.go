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
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_getProjectConfig(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testConfig := &kargoapi.ProjectConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      testProject.Name,
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/config",
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "ProjectConfig does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "gets ProjectConfig",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testConfig,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ProjectConfig in the response
					config := &kargoapi.ProjectConfig{}
					err := json.Unmarshal(w.Body.Bytes(), config)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, config.Namespace)
					require.Equal(t, testProject.Name, config.Name)
				},
			},
		},
	)
}

func Test_server_getProjectConfig_watch(t *testing.T) {
	const projectName = "fake-project"

	testRESTWatchEndpoint(
		t, &config.ServerConfig{},
		"/v1beta1/projects/"+projectName+"/config?watch=true",
		[]restWatchTestCase{
			{
				name:          "project not found",
				url:           "/v1beta1/projects/non-existent/config?watch=true",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "project config not found",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "watches project config successfully",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
					&kargoapi.ProjectConfig{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      projectName,
						},
					},
				),
				operations: func(ctx context.Context, c client.Client) {
					// Update the project config to trigger a watch event
					// Fetch the current config first to get the resource version
					config := &kargoapi.ProjectConfig{}
					_ = c.Get(ctx, client.ObjectKey{Namespace: projectName, Name: projectName}, config)

					config.Spec.PromotionPolicies = []kargoapi.PromotionPolicy{
						{AutoPromotionEnabled: true},
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
