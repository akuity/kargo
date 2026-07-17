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

func Test_server_getClusterPromotionTask(t *testing.T) {
	testTask := &kargoapi.ClusterPromotionTask{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-task"},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/shared/cluster-promotion-tasks/"+testTask.Name,
		[]restTestCase{
			{
				name: "ClusterPromotionTask does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "gets ClusterPromotionTask",
				clientBuilder: fake.NewClientBuilder().WithObjects(testTask),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ClusterPromotionTask in the response
					task := &kargoapi.ClusterPromotionTask{}
					err := json.Unmarshal(w.Body.Bytes(), task)
					require.NoError(t, err)
					require.Equal(t, testTask.Name, task.Name)
				},
			},
		},
	)
}
