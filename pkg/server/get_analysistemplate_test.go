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
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_getAnalysisTemplate(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testTemplate := &rollouts.AnalysisTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-template",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{RolloutsIntegrationEnabled: true},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/analysis-templates/"+testTemplate.Name,
		[]restTestCase{
			{
				name:          "Rollouts integration disabled",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverConfig:  &config.ServerConfig{RolloutsIntegrationEnabled: false},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name:         "Project does not exist",
				serverConfig: &config.ServerConfig{RolloutsIntegrationEnabled: true},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "AnalysisTemplate does not exist",
				serverConfig:  &config.ServerConfig{RolloutsIntegrationEnabled: true},
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:         "gets AnalysisTemplate",
				serverConfig: &config.ServerConfig{RolloutsIntegrationEnabled: true},
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testTemplate,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the AnalysisTemplate in the response
					template := &rollouts.AnalysisTemplate{}
					err := json.Unmarshal(w.Body.Bytes(), template)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, template.Namespace)
					require.Equal(t, testTemplate.Name, template.Name)
				},
			},
		},
	)
}
