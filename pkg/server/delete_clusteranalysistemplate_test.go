package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_deleteClusterAnalysisTemplate(t *testing.T) {
	testTemplate := &rolloutsapi.ClusterAnalysisTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-template"},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{RolloutsIntegrationEnabled: true},
		http.MethodDelete, "/v1beta1/shared/cluster-analysis-templates/"+testTemplate.Name,
		[]restTestCase{
			{
				name:         "Rollouts integration disabled",
				serverConfig: &config.ServerConfig{RolloutsIntegrationEnabled: false},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name: "ClusterAnalysisTemplate does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "deletes ClusterAnalysisTemplate",
				clientBuilder: fake.NewClientBuilder().WithObjects(testTemplate),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					// Verify the ClusterAnalysisTemplate was deleted from the cluster
					template := &rolloutsapi.ClusterAnalysisTemplate{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testTemplate),
						template,
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))
				},
			},
		},
	)
}
