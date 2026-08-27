package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_updateResources(t *testing.T) {
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
		http.MethodPut, "/v1beta1/resources",
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
				body: mustJSONBody(testWarehouse),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "updates resources from JSON",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testWarehouse,
				),
				body: mustJSONArrayBody(
					func() *kargoapi.Project {
						updated := testProject.DeepCopy()
						if updated.Labels == nil {
							updated.Labels = make(map[string]string)
						}
						updated.Labels["updated"] = trueStr
						return updated
					}(),
					func() *kargoapi.Warehouse {
						updated := testWarehouse.DeepCopy()
						if updated.Labels == nil {
							updated.Labels = make(map[string]string)
						}
						updated.Labels["updated"] = trueStr
						return updated
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the response
					var res createOrUpdateResourceResponse
					err := json.Unmarshal(w.Body.Bytes(), &res)
					require.NoError(t, err)
					require.Len(t, res.Results, 2)
					require.Empty(t, res.Results[0].Error)
					require.Empty(t, res.Results[1].Error)

					// Verify the Warehouse was updated in the cluster
					warehouse := &kargoapi.Warehouse{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testWarehouse),
						warehouse,
					)
					require.NoError(t, err)
					require.Equal(t, trueStr, warehouse.Labels["updated"])
				},
			},
			{
				name: "updates resources from YAML",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testWarehouse,
				),
				body: mustYAMLBody(
					func() *kargoapi.Project {
						updated := testProject.DeepCopy()
						if updated.Labels == nil {
							updated.Labels = make(map[string]string)
						}
						updated.Labels["updated"] = trueStr
						return updated
					}(),
					func() *kargoapi.Warehouse {
						updated := testWarehouse.DeepCopy()
						if updated.Labels == nil {
							updated.Labels = make(map[string]string)
						}
						updated.Labels["updated"] = trueStr
						return updated
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the response
					var res createOrUpdateResourceResponse
					err := json.Unmarshal(w.Body.Bytes(), &res)
					require.NoError(t, err)
					require.Len(t, res.Results, 2)
					require.Empty(t, res.Results[0].Error)
					require.Empty(t, res.Results[1].Error)

					// Verify the Project was updated in the cluster
					project := &kargoapi.Project{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testProject),
						project,
					)
					require.NoError(t, err)

					// Verify the Warehouse was updated in the cluster
					warehouse := &kargoapi.Warehouse{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testWarehouse),
						warehouse,
					)
					require.NoError(t, err)
					require.Equal(t, trueStr, warehouse.Labels["updated"])
				},
			},
			{
				name: "upsert creates resources from JSON",
				url:  "/v1beta1/resources?upsert=true",
				body: mustJSONArrayBody(testProject, testWarehouse),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the response
					var res createOrUpdateResourceResponse
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
				name: "upsert creates resources from YAML",
				url:  "/v1beta1/resources?upsert=true",
				body: mustYAMLBody(testProject, testWarehouse),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the response
					var res createOrUpdateResourceResponse
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
				name: "upsert updates existing resources from JSON",
				url:  "/v1beta1/resources?upsert=true",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testWarehouse,
				),
				body: mustJSONArrayBody(
					func() *kargoapi.Project {
						updated := testProject.DeepCopy()
						updated.Labels = map[string]string{"updated": trueStr}
						return updated
					}(),
					func() *kargoapi.Warehouse {
						updated := testWarehouse.DeepCopy()
						updated.Labels = map[string]string{"updated": trueStr}
						return updated
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the response
					var res createOrUpdateResourceResponse
					err := json.Unmarshal(w.Body.Bytes(), &res)
					require.NoError(t, err)
					require.Len(t, res.Results, 2)
					require.Empty(t, res.Results[0].Error)
					require.Nil(t, res.Results[0].CreatedResourceManifest)
					require.NotNil(t, res.Results[0].UpdatedResourceManifest)

					// Examine the Project in the response
					resProject := res.Results[0].UpdatedResourceManifest
					require.Equal(t, testProject.APIVersion, resProject["apiVersion"])
					require.Equal(t, testProject.Kind, resProject["kind"])
					resProjectMeta := resProject["metadata"].(map[string]any) // nolint: forcetypeassert
					require.Equal(t, testProject.Name, resProjectMeta["name"])
					require.Equal(t, trueStr, resProjectMeta["labels"].(map[string]any)["updated"]) // nolint: forcetypeassert

					// Examine the Warehouse in the response
					resWarehouse := res.Results[1].UpdatedResourceManifest
					require.Equal(t, testWarehouse.APIVersion, resWarehouse["apiVersion"])
					require.Equal(t, testWarehouse.Kind, resWarehouse["kind"])
					resWarehouseMeta := resWarehouse["metadata"].(map[string]any) // nolint: forcetypeassert
					require.Equal(t, testWarehouse.Name, resWarehouseMeta["name"])
					require.Equal(t, testWarehouse.Namespace, resWarehouseMeta["namespace"])
					require.Equal(t, trueStr, resWarehouseMeta["labels"].(map[string]any)["updated"]) // nolint: forcetypeassert

					// Verify the Project in the cluster was modified
					project := &kargoapi.Project{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testProject),
						project,
					)
					require.NoError(t, err)
					require.Equal(t, trueStr, project.Labels["updated"])

					// Verify the Warehouse in the cluster was modified
					warehouse := &kargoapi.Warehouse{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testWarehouse),
						warehouse,
					)
					require.NoError(t, err)
					require.Equal(t, trueStr, warehouse.Labels["updated"])
				},
			},
			{
				name: "upsert updates existing resources from YAML",
				url:  "/v1beta1/resources?upsert=true",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testWarehouse,
				),
				body: mustYAMLBody(
					func() *kargoapi.Project {
						updated := testProject.DeepCopy()
						updated.Labels = map[string]string{"updated": trueStr}
						return updated
					}(),
					func() *kargoapi.Warehouse {
						updated := testWarehouse.DeepCopy()
						updated.Labels = map[string]string{"updated": trueStr}
						return updated
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the response
					var res createOrUpdateResourceResponse
					err := json.Unmarshal(w.Body.Bytes(), &res)
					require.NoError(t, err)
					require.Len(t, res.Results, 2)
					require.Empty(t, res.Results[0].Error)
					require.Empty(t, res.Results[1].Error)
					require.Nil(t, res.Results[0].CreatedResourceManifest)
					require.NotNil(t, res.Results[0].UpdatedResourceManifest)
					require.Nil(t, res.Results[1].CreatedResourceManifest)
					require.NotNil(t, res.Results[1].UpdatedResourceManifest)

					// Examine the Project in the response
					resProject := res.Results[0].UpdatedResourceManifest
					require.Equal(t, testProject.APIVersion, resProject["apiVersion"])
					require.Equal(t, testProject.Kind, resProject["kind"])
					resProjectMeta := resProject["metadata"].(map[string]any) // nolint: forcetypeassert
					require.Equal(t, testProject.Name, resProjectMeta["name"])
					require.Equal(t, trueStr, resProjectMeta["labels"].(map[string]any)["updated"]) // nolint: forcetypeassert

					// Examine the Warehouse in the response
					resWarehouse := res.Results[1].UpdatedResourceManifest
					require.Equal(t, testWarehouse.APIVersion, resWarehouse["apiVersion"])
					require.Equal(t, testWarehouse.Kind, resWarehouse["kind"])
					resWarehouseMeta := resWarehouse["metadata"].(map[string]any) // nolint: forcetypeassert
					require.Equal(t, testWarehouse.Name, resWarehouseMeta["name"])
					require.Equal(t, testWarehouse.Namespace, resWarehouseMeta["namespace"])
					require.Equal(t, trueStr, resWarehouseMeta["labels"].(map[string]any)["updated"]) // nolint: forcetypeassert

					// Verify the Project in the cluster was modified
					project := &kargoapi.Project{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testProject),
						project,
					)
					require.NoError(t, err)
					require.Equal(t, trueStr, project.Labels["updated"])

					// Verify the Warehouse in the cluster was modified
					warehouse := &kargoapi.Warehouse{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testWarehouse),
						warehouse,
					)
					require.NoError(t, err)
					require.Equal(t, trueStr, warehouse.Labels["updated"])
				},
			},
			{
				name: "partial failure",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
				),
				body: mustJSONArrayBody(
					testProject,
					testWarehouse, // Does not already exist and upsert is not set; should fail
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the response
					var res createOrUpdateResourceResponse
					err := json.Unmarshal(w.Body.Bytes(), &res)
					require.NoError(t, err)
					require.Len(t, res.Results, 2)

					// First result (Project) should succeed
					require.Empty(t, res.Results[0].Error)

					// Second result (Warehouse) should have error
					require.NotNil(t, res.Results[1].Error)
					require.Contains(t, res.Results[1].Error, "does not exist")

					// Verify the Project was updated in the cluster
					project := &kargoapi.Project{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testProject),
						project,
					)
					require.NoError(t, err)
				},
			},
		},
	)
}

// Test_server_updateResources_clusterScopedNamespaceSpoofing is the
// updateResources (PUT .../resources?upsert=true) counterpart to
// Test_server_createResources_clusterScopedNamespaceSpoofing in
// create_resource_v1alpha1_test.go; see that test's doc comment for the full
// explanation. newProjectCreatorClient is defined there and reused here.
func Test_server_updateResources_clusterScopedNamespaceSpoofing(t *testing.T) {
	const projectName = "ghsa-repro"

	// capturedClient is reassigned by each serverSetup before assertions reads
	// it. Test cases are run sequentially, so this is safe.
	var capturedClient client.WithWatch

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPut, "/v1beta1/resources?upsert=true",
		[]restTestCase{
			{
				name: "control: direct upsert of a cluster-scoped resource is denied",
				serverSetup: func(t *testing.T, s *server) {
					s.client, capturedClient = newProjectCreatorClient(t, fake.NewClientBuilder())
					s.authorizeFn = s.client.Authorize
				},
				body: mustJSONBody(&kargoapi.ClusterPromotionTask{
					TypeMeta: metav1.TypeMeta{
						APIVersion: kargoapi.GroupVersion.String(),
						Kind:       "ClusterPromotionTask",
					},
					ObjectMeta: metav1.ObjectMeta{Name: "control-task"},
					Spec: kargoapi.PromotionTaskSpec{
						Steps: []kargoapi.PromotionStep{{Uses: "fail"}},
					},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
				},
			},
			{
				name: "exploit: ClusterPromotionTask namespace spoofed to a just-created Project is still denied",
				serverSetup: func(t *testing.T, s *server) {
					s.client, capturedClient = newProjectCreatorClient(t, fake.NewClientBuilder())
					s.authorizeFn = s.client.Authorize
				},
				body: mustJSONArrayBody(
					&kargoapi.Project{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kargoapi.GroupVersion.String(),
							Kind:       "Project",
						},
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
					&kargoapi.ClusterPromotionTask{
						TypeMeta: metav1.TypeMeta{
							APIVersion: kargoapi.GroupVersion.String(),
							Kind:       "ClusterPromotionTask",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "exploit-task",
							// The exploit: meaningless to Kubernetes for a cluster-scoped kind.
							Namespace: projectName,
						},
						Spec: kargoapi.PromotionTaskSpec{
							Steps: []kargoapi.PromotionStep{{Uses: "fail"}},
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					var res createOrUpdateResourceResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
					require.Len(t, res.Results, 2)
					require.Empty(t, res.Results[0].Error, "Project creation should succeed")
					require.NotEmpty(
						t, res.Results[1].Error,
						"ClusterPromotionTask creation must be denied, not silently bypassed via a spoofed namespace",
					)
					require.Contains(t, res.Results[1].Error, "forbidden")

					err := capturedClient.Get(
						t.Context(),
						client.ObjectKey{Name: "exploit-task"},
						&kargoapi.ClusterPromotionTask{},
					)
					require.True(t, apierrors.IsNotFound(err), "ClusterPromotionTask must not have been persisted")
				},
			},
		},
	)
}
