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

	rollouts "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_getClusterAnalysisTemplate(t *testing.T) {
	testTemplate := &rollouts.ClusterAnalysisTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-template"},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{RolloutsIntegrationEnabled: true},
		http.MethodGet, "/v1beta1/shared/cluster-analysis-templates/"+testTemplate.Name,
		[]restTestCase{
			{
				name:         "Rollouts integration disabled",
				serverConfig: &config.ServerConfig{RolloutsIntegrationEnabled: false},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name: "ClusterAnalysis template does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "gets ClusterAnalysisTemplate",
				clientBuilder: fake.NewClientBuilder().WithObjects(testTemplate),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ClusterAnalysisTemplate in the response
					template := &rollouts.ClusterAnalysisTemplate{}
					err := json.Unmarshal(w.Body.Bytes(), template)
					require.NoError(t, err)
					require.Equal(t, testTemplate.Name, template.Name)
				},
			},
		},
	)
}
