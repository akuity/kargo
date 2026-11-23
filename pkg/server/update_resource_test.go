package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	authnv1 "k8s.io/api/authentication/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/user"
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
				name: "Targets are not Kubernetes resources",
				// A Target manifest, as a user would hand to kargo apply; the Go
				// type carries no TypeMeta since Targets are not custom resources.
				body: mustJSONBody(map[string]any{
					"apiVersion": kargoapi.GroupVersion.String(),
					"kind":       "Target",
					"metadata":   map[string]any{"namespace": testProject.Name, "name": "us-east-1"},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
					require.Contains(t, w.Body.String(), "/v1beta1/projects/{project}/targets")
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
				name: "upsert denies Promotion creation without promote permission",
				url:  "/v1beta1/resources?upsert=true",
				serverSetup: func(_ *testing.T, s *server) {
					s.authorizeFn = func(
						_ context.Context,
						_ string,
						gvr schema.GroupVersionResource,
						_ string,
						key client.ObjectKey,
					) error {
						return apierrors.NewForbidden(
							gvr.GroupResource(),
							key.Name,
							errors.New("not permitted to promote"),
						)
					}
				},
				body: mustJSONBody(&kargoapi.Promotion{
					TypeMeta: metav1.TypeMeta{
						APIVersion: kargoapi.GroupVersion.String(),
						Kind:       "Promotion",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-promotion",
						Namespace: testProject.Name,
					},
					Spec: kargoapi.PromotionSpec{
						Stage:   "fake-stage",
						Freight: "fake-freight",
					},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
					// The Promotion must not have been created.
					err := c.Get(
						t.Context(),
						client.ObjectKey{Namespace: testProject.Name, Name: "fake-promotion"},
						&kargoapi.Promotion{},
					)
					require.True(t, apierrors.IsNotFound(err))
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
// create_resource_test.go; see that test's doc comment for the full
// explanation.
func Test_server_updateResources_clusterScopedNamespaceSpoofing(t *testing.T) {
	const projectName = "ghsa-mmqq-repro"

	projectCreatorUser := user.KubernetesUser{
		UserInfo: authnv1.UserInfo{
			Username: "system:serviceaccount:kargo:kargo-project-creator",
		},
	}

	// capturedClient is reassigned by each serverSetup before assertions reads
	// it. Test cases are run sequentially, so this is safe (see the same
	// pattern used for the freight event sender in create_resource_test.go).
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
				ctxSetup: func(ctx context.Context) context.Context {
					return user.ContextWithIdentity(ctx, projectCreatorUser)
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
				ctxSetup: func(ctx context.Context) context.Context {
					return user.ContextWithIdentity(ctx, projectCreatorUser)
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

func Test_server_updateResources_projectConfig(t *testing.T) {
	existingProject := testResourceProject("existing-project", "existing-uid")
	newProject := testResourceProject("new-project", "")

	// store is reassigned by each serverSetup before assertions reads it.
	// Test cases are run sequentially, so this is safe.
	var store *fakeProjectConfigStore

	testRESTEndpoint(
		t, nil,
		http.MethodPut, "/v1beta1/resources",
		[]restTestCase{
			{
				name:          "updates a ProjectConfig in the database",
				clientBuilder: fake.NewClientBuilder().WithObjects(existingProject),
				serverSetup:   withProjectConfigStore(storeHolding(existingProject), &store, nil),
				body:          mustJSONBody(testResourceProjectConfig(existingProject.Name)),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var res createOrUpdateResourceResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
					require.Len(t, res.Results, 1)
					require.Empty(t, res.Results[0].Error)
					require.Nil(t, res.Results[0].CreatedResourceManifest)
					manifest := res.Results[0].UpdatedResourceManifest
					require.Equal(t, kargoapi.GroupVersion.String(), manifest["apiVersion"])
					require.Equal(t, "ProjectConfig", manifest["kind"])

					stored := store.configs[existingProject.Name]
					require.Equal(t, int64(2), stored.Generation)
					require.Contains(t, string(stored.Spec), "https://example.com")
					requireNoProjectConfigInKubernetes(t, c, existingProject.Name)
				},
			},
			{
				name:          "ProjectConfig does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(existingProject),
				serverSetup:   withProjectConfigStore(&fakeProjectConfigStore{}, &store, nil),
				body:          mustJSONBody(testResourceProjectConfig(existingProject.Name)),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
					require.Empty(t, store.configs)
				},
			},
			{
				name:          "upsert creates a ProjectConfig in the database",
				url:           "/v1beta1/resources?upsert=true",
				clientBuilder: fake.NewClientBuilder().WithObjects(existingProject),
				serverSetup:   withProjectConfigStore(&fakeProjectConfigStore{}, &store, nil),
				body:          mustYAMLBody(testResourceProjectConfig(existingProject.Name)),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var res createOrUpdateResourceResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
					require.Len(t, res.Results, 1)
					require.Empty(t, res.Results[0].Error)
					require.Nil(t, res.Results[0].UpdatedResourceManifest)
					require.Equal(t, "ProjectConfig", res.Results[0].CreatedResourceManifest["kind"])

					stored, ok := store.configs[existingProject.Name]
					require.True(t, ok)
					require.Equal(t, string(existingProject.UID), stored.ProjectUID)
					requireNoProjectConfigInKubernetes(t, c, existingProject.Name)
				},
			},
			{
				name:        "Project does not exist",
				url:         "/v1beta1/resources?upsert=true",
				serverSetup: withProjectConfigStore(&fakeProjectConfigStore{}, &store, nil),
				body:        mustJSONBody(testResourceProjectConfig(existingProject.Name)),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
					require.Empty(t, store.configs)
				},
			},
			{
				name:          "not authorized",
				url:           "/v1beta1/resources?upsert=true",
				clientBuilder: fake.NewClientBuilder().WithObjects(existingProject),
				serverSetup: withProjectConfigStore(
					storeHolding(existingProject),
					&store,
					forbidProjectConfigs,
				),
				body: mustJSONBody(testResourceProjectConfig(existingProject.Name)),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusForbidden, w.Code)
					require.Equal(t, int64(1), store.configs[existingProject.Name].Generation)
				},
			},
			{
				name: "caller with no access to Projects updates a ProjectConfig",
				serverSetup: withClientForbiddingProjects(
					storeHolding(existingProject),
					&store,
					existingProject,
				),
				ctxSetup: asProjectScopedUser,
				body:     mustJSONBody(testResourceProjectConfig(existingProject.Name)),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					require.Equal(t, int64(2), store.configs[existingProject.Name].Generation)
				},
			},
			{
				name:          "wizard upserts a Project and its ProjectConfig",
				url:           "/v1beta1/resources?upsert=true",
				clientBuilder: assigningProjectUIDs(),
				// The caller has no permissions on ProjectConfigs yet: the
				// management controller grants them asynchronously.
				serverSetup: withProjectConfigStore(
					&fakeProjectConfigStore{},
					&store,
					forbidProjectConfigs,
				),
				body: mustJSONArrayBody(newProject, testResourceProjectConfig(newProject.Name)),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var res createOrUpdateResourceResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
					require.Len(t, res.Results, 2)
					require.Empty(t, res.Results[0].Error)
					require.Empty(t, res.Results[1].Error)

					stored, ok := store.configs[newProject.Name]
					require.True(t, ok)
					require.Equal(t, createdProjectUID, stored.ProjectUID)
				},
			},
			{
				name:          "wizard exception does not cover a Project that already existed",
				url:           "/v1beta1/resources?upsert=true",
				clientBuilder: fake.NewClientBuilder().WithObjects(existingProject),
				serverSetup: withProjectConfigStore(
					&fakeProjectConfigStore{},
					&store,
					forbidProjectConfigs,
				),
				// Updating the existing Project does not create it.
				body: mustJSONArrayBody(
					existingProject,
					testResourceProjectConfig(existingProject.Name),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					var res createOrUpdateResourceResponse
					require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
					require.Len(t, res.Results, 2)
					require.Empty(t, res.Results[0].Error)
					require.NotNil(t, res.Results[0].UpdatedResourceManifest)
					require.Contains(t, res.Results[1].Error, "forbidden")
					require.Empty(t, store.configs)
				},
			},
		},
	)
}
