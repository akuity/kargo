package server

import (
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

func Test_server_listPromotionRequests(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}

	newPromotionRequest := func(name, stage string) *kargoapi.PromotionRequest {
		return &kargoapi.PromotionRequest{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testProject.Name,
				Name:      name,
			},
			Spec: kargoapi.PromotionRequestSpec{
				Stage:   stage,
				Freight: "fake-freight",
			},
		}
	}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/promotion-requests",
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "no PromotionRequests exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &kargoapi.PromotionRequestList{}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), list))
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists PromotionRequests",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					newPromotionRequest("promotion-request-2", "fake-stage"),
					newPromotionRequest("promotion-request-1", "fake-stage"),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &kargoapi.PromotionRequestList{}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), list))
					require.Len(t, list.Items, 2)
					// Sorted ascending by name, which -- because names embed a
					// ULID -- is also creation order.
					require.Equal(t, "promotion-request-1", list.Items[0].Name)
					require.Equal(t, "promotion-request-2", list.Items[1].Name)
				},
			},
			{
				name: "filters by Stage",
				url:  "/v1beta1/projects/" + testProject.Name + "/promotion-requests?stage=wanted-stage",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					newPromotionRequest("promotion-request-1", "wanted-stage"),
					newPromotionRequest("promotion-request-2", "other-stage"),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &kargoapi.PromotionRequestList{}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), list))
					require.Len(t, list.Items, 1)
					require.Equal(t, "promotion-request-1", list.Items[0].Name)
				},
			},
		},
	)
}
