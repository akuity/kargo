package server

import (
	"bytes"
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

func Test_server_deleteResources(t *testing.T) {
	testProject := &kargoapi.Project{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Project",
		},
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testWarehouse := &kargoapi.Warehouse{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Warehouse",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-warehouse",
			Namespace: testProject.Name,
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodDelete, "/v1beta1/resources",
		[]restTestCase{
			{
				name: "empty request body",
				body: bytes.NewBufferString(""),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "invalid JSON in request body",
				body: bytes.NewBufferString("{invalid json"),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "invalid YAML in request body",
				body: bytes.NewBufferString("invalid: [unclosed sequence"),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "resource does not exist",
				body: mustJSONBody(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "deletes resources from JSON",
				body: mustJSONArrayBody(testWarehouse, testProject),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testWarehouse,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the response
					var res deleteResourceResponse
					err := json.Unmarshal(w.Body.Bytes(), &res)
					require.NoError(t, err)
					require.Len(t, res.Results, 2)
					require.Empty(t, res.Results[0].Error)
					require.Empty(t, res.Results[1].Error)

					// Verify the Warehouse was deleted from the cluster
					warehouse := &kargoapi.Warehouse{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testWarehouse),
						warehouse,
					)
					require.Error(t, err)
					require.True(t, client.IgnoreNotFound(err) == nil)

					// Verify the Project was deleted from the cluster
					project := &kargoapi.Project{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testProject),
						project,
					)
					require.Error(t, err)
					require.True(t, client.IgnoreNotFound(err) == nil)
				},
			},
			{
				name: "deletes resources from YAML",
				body: mustYAMLBody(testWarehouse, testProject),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testWarehouse,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the response
					var res deleteResourceResponse
					err := json.Unmarshal(w.Body.Bytes(), &res)
					require.NoError(t, err)
					require.Len(t, res.Results, 2)
					require.Empty(t, res.Results[0].Error)
					require.Empty(t, res.Results[1].Error)

					// Verify the Warehouse was deleted from the cluster
					warehouse := &kargoapi.Warehouse{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testWarehouse),
						warehouse,
					)
					require.Error(t, err)
					require.True(t, client.IgnoreNotFound(err) == nil)

					// Verify the Project was deleted from the cluster
					project := &kargoapi.Project{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testProject),
						project,
					)
					require.Error(t, err)
					require.True(t, client.IgnoreNotFound(err) == nil)
				},
			},
			{
				name: "partial failure",
				body: mustJSONArrayBody(testWarehouse, testProject),
				// Only testProject exists, testWarehouse does not
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the response
					var res deleteResourceResponse
					err := json.Unmarshal(w.Body.Bytes(), &res)
					require.NoError(t, err)
					require.Len(t, res.Results, 2)

					// First result (Warehouse) should have error
					require.NotEmpty(t, res.Results[0].Error)
					require.Contains(t, res.Results[0].Error, "not found")

					// Second result (Project) should succeed
					require.Empty(t, res.Results[1].Error)
					require.NotNil(t, res.Results[1].DeletedResourceManifest)

					// Verify the Project was deleted from the cluster
					project := &kargoapi.Project{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testProject),
						project,
					)
					require.Error(t, err)
					require.True(t, client.IgnoreNotFound(err) == nil)
				},
			},
		},
	)
}
