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

func Test_server_getTarget(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testTarget := &kargoapi.Target{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "us-east-1",
			Labels:    map[string]string{"region": "us"},
		},
	}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/targets/"+testTarget.Name,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Target does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "returns the Target",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testTarget),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					target := &kargoapi.Target{}
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), target))
					require.Equal(t, testTarget.Name, target.Name)
					require.Equal(t, testTarget.Labels, target.Labels)
				},
			},
		},
	)
}
