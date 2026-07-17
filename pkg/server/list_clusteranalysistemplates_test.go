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

func Test_server_listClusterAnalysisTemplates(t *testing.T) {
	testRESTEndpoint(
		t, &config.ServerConfig{RolloutsIntegrationEnabled: true},
		http.MethodGet, "/v1beta1/shared/cluster-analysis-templates",
		[]restTestCase{
			{
				name:         "Rollouts integration disabled",
				serverConfig: &config.ServerConfig{RolloutsIntegrationEnabled: false},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name: "no ClusterAnalysisTemplates exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					templates := &rollouts.ClusterAnalysisTemplateList{}
					err := json.Unmarshal(w.Body.Bytes(), templates)
					require.NoError(t, err)
					require.Empty(t, templates.Items)
				},
			},
			{
				name: "lists ClusterAnalysisTemplates",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&rollouts.ClusterAnalysisTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name: "template-1",
						},
					},
					&rollouts.ClusterAnalysisTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name: "template-2",
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ClusterAnalysisTemplates in the response
					templates := &rollouts.ClusterAnalysisTemplateList{}
					err := json.Unmarshal(w.Body.Bytes(), templates)
					require.NoError(t, err)
					require.Len(t, templates.Items, 2)
				},
			},
		},
	)
}
