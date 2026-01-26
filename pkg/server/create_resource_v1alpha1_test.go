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
)

func Test_server_createResources(t *testing.T) {
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
		t, nil,
		http.MethodPost, "/v1beta1/resources",
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
				name:          "resource already exists",
				body:          mustJSONBody(testProject),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "creates resources from JSON",
				body: mustJSONArrayBody(testProject, testWarehouse),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					// Examine the response
					var res createResourceResponse
					err := json.Unmarshal(w.Body.Bytes(), &res)
					require.NoError(t, err)
					require.Len(t, res.Results, 2)
					require.Empty(t, res.Results[0].Error)
					require.Empty(t, res.Results[1].Error)

					// Examine the Project in the response
					resProject := res.Results[0].CreatedResourceManifest
					require.Equal(t, testProject.APIVersion, resProject["apiVersion"])
					require.Equal(t, testProject.Kind, resProject["kind"])
					resProjectMeta := resProject["metadata"].(map[string]any) // nolint: forcetypeassert
					require.Equal(t, testProject.Name, resProjectMeta["name"])

					// Examine the Warehouse in the response
					resWarehouse := res.Results[1].CreatedResourceManifest
					require.Equal(t, testWarehouse.APIVersion, resWarehouse["apiVersion"])
					require.Equal(t, testWarehouse.Kind, resWarehouse["kind"])
					resWarehouseMeta := resWarehouse["metadata"].(map[string]any) // nolint: forcetypeassert
					require.Equal(t, testWarehouse.Name, resWarehouseMeta["name"])
					require.Equal(t, testWarehouse.Namespace, resWarehouseMeta["namespace"])

					// Verify the Project was created in the cluster
					project := &kargoapi.Project{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testProject),
						project,
					)
					require.NoError(t, err)

					// Verify the Warehouse was created in the cluster
					warehouse := &kargoapi.Warehouse{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testWarehouse),
						warehouse,
					)
					require.NoError(t, err)
				},
			},
			{
				name: "creates resources from YAML",
				body: mustYAMLBody(testProject, testWarehouse),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					// Examine the response
					var res createResourceResponse
					err := json.Unmarshal(w.Body.Bytes(), &res)
					require.NoError(t, err)
					require.Len(t, res.Results, 2)
					require.Empty(t, res.Results[0].Error)
					require.Empty(t, res.Results[1].Error)

					// Examine the Project in the response
					resProject := res.Results[0].CreatedResourceManifest
					require.Equal(t, testProject.APIVersion, resProject["apiVersion"])
					require.Equal(t, testProject.Kind, resProject["kind"])
					resProjectMeta := resProject["metadata"].(map[string]any) // nolint: forcetypeassert
					require.Equal(t, testProject.Name, resProjectMeta["name"])

					// Examine the Warehouse in the response
					resWarehouse := res.Results[1].CreatedResourceManifest
					require.Equal(t, testWarehouse.APIVersion, resWarehouse["apiVersion"])
					require.Equal(t, testWarehouse.Kind, resWarehouse["kind"])
					resWarehouseMeta := resWarehouse["metadata"].(map[string]any) // nolint: forcetypeassert
					require.Equal(t, testWarehouse.Name, resWarehouseMeta["name"])
					require.Equal(t, testWarehouse.Namespace, resWarehouseMeta["namespace"])

					// Verify the Project was created in the cluster
					project := &kargoapi.Project{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testProject),
						project,
					)
					require.NoError(t, err)

					// Verify the Warehouse was created in the cluster
					warehouse := &kargoapi.Warehouse{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testWarehouse),
						warehouse,
					)
					require.NoError(t, err)
				},
			},
		},
	)
}
