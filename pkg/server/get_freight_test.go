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

func Test_server_getFreight(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testFreight := &kargoapi.Freight{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-freight",
			Labels: map[string]string{
				kargoapi.LabelKeyAlias: "fake-alias",
			},
		},
		Alias: "fake-alias",
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/freight/"+testFreight.Name,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Freight does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "gets Freight by name",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Freight in the response
					freight := &kargoapi.Freight{}
					err := json.Unmarshal(w.Body.Bytes(), freight)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, freight.Namespace)
					require.Equal(t, testFreight.Name, freight.Name)
				},
			},
			{
				name: "gets Freight by alias",
				url:  "/v1beta1/projects/" + testProject.Name + "/freight/" + testFreight.Alias,
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testFreight,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Freight in the response
					freight := &kargoapi.Freight{}
					err := json.Unmarshal(w.Body.Bytes(), freight)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, freight.Namespace)
					require.Equal(t, testFreight.Alias, freight.Alias)
				},
			},
		},
	)
}
