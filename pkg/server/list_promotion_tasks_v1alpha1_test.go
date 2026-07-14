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

func Test_server_listPromotionTasks(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/promotion-tasks",
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "no PromotionTasks exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &kargoapi.PromotionTaskList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists PromotionTasks",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					&kargoapi.PromotionTask{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "task-1",
						},
					},
					&kargoapi.PromotionTask{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "task-2",
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the PromotionTasks in the response
					tasks := &kargoapi.PromotionTaskList{}
					err := json.Unmarshal(w.Body.Bytes(), tasks)
					require.NoError(t, err)
					require.Len(t, tasks.Items, 2)
				},
			},
		},
	)
}
