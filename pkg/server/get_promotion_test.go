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

func Test_server_getPromotion(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testPromo := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-promotion",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/promotions/"+testPromo.Name,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Promotion does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "gets Promotion",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testPromo,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Promotion in the response
					promo := &kargoapi.Promotion{}
					err := json.Unmarshal(w.Body.Bytes(), promo)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, promo.Namespace)
					require.Equal(t, testPromo.Name, promo.Name)
				},
			},
		},
	)
}

func Test_server_getPromotion_watch(t *testing.T) {
	const projectName = "fake-project"
	const promotionName = "fake-promotion"

	testRESTWatchEndpoint(
		t, &config.ServerConfig{},
		"/v1beta1/projects/"+projectName+"/promotions/"+promotionName+"?watch=true",
		[]restWatchTestCase{
			{
				name: "promotion not found",
				url:  "/v1beta1/projects/" + projectName + "/promotions/non-existent?watch=true",
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
				name: "watches promotion successfully",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
					&kargoapi.Promotion{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      promotionName,
						},
					},
				),
				operations: func(ctx context.Context, c client.Client) {
					// Fetch the current promotion first to get the resource version
					promo := &kargoapi.Promotion{}
					_ = c.Get(ctx, client.ObjectKey{Namespace: projectName, Name: promotionName}, promo)

					// Update the promotion to trigger a watch event
					promo.Spec.Stage = "test-stage"
					_ = c.Update(ctx, promo)
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
