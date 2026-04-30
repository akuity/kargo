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

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_server_deleteProjectConfig(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testConfig := &kargoapi.ProjectConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      testProject.Name,
		},
	}
	testRESTEndpoint(
		t, nil,
		http.MethodDelete, "/v1beta1/projects/"+testProject.Name+"/config",
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "ProjectConfig does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "deletes ProjectConfig",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testConfig,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					// Verify the ProjectConfig was deleted from the cluster
					config := &kargoapi.ProjectConfig{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testConfig),
						config,
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))
				},
			},
		},
	)
}
