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

func Test_server_listClusterPromotionTasks(t *testing.T) {
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/shared/cluster-promotion-tasks",
		[]restTestCase{
			{
				name: "no ClusterPromotionTasks exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &kargoapi.ClusterPromotionTaskList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists ClusterPromotionTasks",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.ClusterPromotionTask{
						ObjectMeta: metav1.ObjectMeta{
							Name: "task-1",
						},
					},
					&kargoapi.ClusterPromotionTask{
						ObjectMeta: metav1.ObjectMeta{
							Name: "task-2",
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ClusterPromotionTasks in the response
					tasks := &kargoapi.ClusterPromotionTaskList{}
					err := json.Unmarshal(w.Body.Bytes(), tasks)
					require.NoError(t, err)
					require.Len(t, tasks.Items, 2)
				},
			},
		},
	)
}
