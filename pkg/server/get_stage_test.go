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

func Test_server_getStage(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testStage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-stage",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/stages/"+testStage.Name,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Stage does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "gets Stage",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					&kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      testStage.Name,
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Stage in the response
					stage := &kargoapi.Stage{}
					err := json.Unmarshal(w.Body.Bytes(), stage)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, stage.Namespace)
					require.Equal(t, testStage.Name, stage.Name)
				},
			},
		},
	)
}

func Test_server_getStage_watch(t *testing.T) {
	const projectName = "fake-project"
	const stageName = "fake-stage"

	testRESTWatchEndpoint(
		t, &config.ServerConfig{},
		"/v1beta1/projects/"+projectName+"/stages/"+stageName+"?watch=true",
		[]restWatchTestCase{
			{
				name: "stage not found",
				url:  "/v1beta1/projects/" + projectName + "/stages/non-existent?watch=true",
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
				name: "watches stage successfully",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
					&kargoapi.Stage{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      stageName,
						},
					},
				),
				operations: func(ctx context.Context, c client.Client) {
					// Fetch the current stage first to get the resource version
					stage := &kargoapi.Stage{}
					_ = c.Get(ctx, client.ObjectKey{Namespace: projectName, Name: stageName}, stage)

					// Update the stage to trigger a watch event
					stage.Spec.RequestedFreight = []kargoapi.FreightRequest{{Origin: kargoapi.FreightOrigin{Kind: "Warehouse"}}}
					_ = c.Update(ctx, stage)
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
