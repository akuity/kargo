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
)

func Test_server_getAnalysisRunRequest(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testReq := &kargoapi.AnalysisRunRequest{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-analysisrunrequest",
		},
		Spec: kargoapi.AnalysisRunRequestSpec{StageName: "fake-stage"},
	}
	testRESTEndpoint(
		t, nil,
		http.MethodGet,
		"/v1beta1/projects/"+testProject.Name+"/analysis-run-requests/"+testReq.Name,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "AnalysisRunRequest does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "gets AnalysisRunRequest",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testReq,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					req := &kargoapi.AnalysisRunRequest{}
					err := json.Unmarshal(w.Body.Bytes(), req)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, req.Namespace)
					require.Equal(t, testReq.Name, req.Name)
					require.Equal(t, testReq.Spec.StageName, req.Spec.StageName)
				},
			},
		},
	)
}
