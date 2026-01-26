package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestUpdateConfigMap(t *testing.T) {
	testCases := map[string]struct {
		req        *svcv1alpha1.UpdateConfigMapRequest
		objects    []client.Object
		assertions func(*testing.T, *connect.Response[svcv1alpha1.UpdateConfigMapResponse], error)
	}{
		"empty name": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name: "",
				Data: map[string]string{"key": "value"},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"empty data": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name: "test-cm",
				Data: map[string]string{},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Contains(t, err.Error(), "ConfigMap data cannot be empty")
				require.Nil(t, r)
			},
		},
		"nil data": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name: "test-cm",
				Data: nil,
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
				require.Contains(t, err.Error(), "ConfigMap data cannot be empty")
				require.Nil(t, r)
			},
		},
		"non-existing project": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name:    "test-cm",
				Project: "non-existing-project",
				Data:    map[string]string{"key": "value"},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
				require.Nil(t, r)
			},
		},
		"update in project namespace": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name:        "cm-1",
				Project:     "kargo-demo",
				Data:        map[string]string{"updated": "data"},
				Description: "updated description",
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				mustNewObject[corev1.ConfigMap]("testdata/config-map-1.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.ConfigMap)
				assert.Equal(t, "cm-1", r.Msg.ConfigMap.Name)
				assert.Equal(t, "kargo-demo", r.Msg.ConfigMap.Namespace)
				assert.Equal(t, map[string]string{"updated": "data"}, r.Msg.ConfigMap.Data)
				assert.Equal(t, "updated description", r.Msg.ConfigMap.Annotations[kargoapi.AnnotationKeyDescription])
			},
		},
		"update in shared namespace": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name:    "shared-cm",
				Project: "",
				Data:    map[string]string{"updated-shared": "data"},
			},
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "shared-cm",
						Namespace: "kargo-shared-resources",
					},
					Data: map[string]string{"old": "data"},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.ConfigMap)
				assert.Equal(t, "shared-cm", r.Msg.ConfigMap.Name)
				assert.Equal(t, "kargo-shared-resources", r.Msg.ConfigMap.Namespace)
				assert.Equal(t, map[string]string{"updated-shared": "data"}, r.Msg.ConfigMap.Data)
			},
		},
		"update system-level": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				SystemLevel: true,
				Name:        "system-cm",
				Data:        map[string]string{"updated-system": "config"},
			},
			objects: []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "system-cm",
						Namespace: "kargo-system-resources",
					},
					Data: map[string]string{"old-system": "config"},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.ConfigMap)
				assert.Equal(t, "system-cm", r.Msg.ConfigMap.Name)
				assert.Equal(t, "kargo-system-resources", r.Msg.ConfigMap.Namespace)
				assert.Equal(t, map[string]string{"updated-system": "config"}, r.Msg.ConfigMap.Data)
			},
		},
		"update non-existing ConfigMap": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name:    "non-existing-cm",
				Project: "kargo-demo",
				Data:    map[string]string{"new": "data"},
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "update configmap")
				require.Nil(t, r)
			},
		},
		"update with multiple data keys": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name:    "cm-1",
				Project: "kargo-demo",
				Data: map[string]string{
					"newKey1": "newValue1",
					"newKey2": "newValue2",
					"newKey3": "newValue3",
				},
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				mustNewObject[corev1.ConfigMap]("testdata/config-map-1.yaml"),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.ConfigMap)
				assert.Equal(t, "cm-1", r.Msg.ConfigMap.Name)
				assert.Len(t, r.Msg.ConfigMap.Data, 3)
				assert.Equal(t, "newValue1", r.Msg.ConfigMap.Data["newKey1"])
				assert.Equal(t, "newValue2", r.Msg.ConfigMap.Data["newKey2"])
				assert.Equal(t, "newValue3", r.Msg.ConfigMap.Data["newKey3"])
			},
		},
		"update clears old data and sets new": {
			req: &svcv1alpha1.UpdateConfigMapRequest{
				Name:    "multi-key-cm",
				Project: "kargo-demo",
				Data:    map[string]string{"onlyKey": "onlyValue"},
			},
			objects: []client.Object{
				mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multi-key-cm",
						Namespace: "kargo-demo",
					},
					Data: map[string]string{
						"oldKey1": "oldValue1",
						"oldKey2": "oldValue2",
					},
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.UpdateConfigMapResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.NotNil(t, r.Msg.ConfigMap)
				assert.Equal(t, "multi-key-cm", r.Msg.ConfigMap.Name)
				assert.Len(t, r.Msg.ConfigMap.Data, 1)
				assert.Equal(t, "onlyValue", r.Msg.ConfigMap.Data["onlyKey"])
				// Verify old keys are not present
				_, hasOldKey1 := r.Msg.ConfigMap.Data["oldKey1"]
				_, hasOldKey2 := r.Msg.ConfigMap.Data["oldKey2"]
				assert.False(t, hasOldKey1)
				assert.False(t, hasOldKey2)
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			client, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						_ context.Context,
						_ *rest.Config,
						scheme *runtime.Scheme,
					) (client.WithWatch, error) {
						c := fake.NewClientBuilder().WithScheme(scheme)
						if len(testCase.objects) > 0 {
							c.WithObjects(testCase.objects...)
						}
						return c.Build(), nil
					},
				},
			)
			require.NoError(t, err)

			svr := &server{
				client: client,
				cfg: config.ServerConfig{
					SharedResourcesNamespace: "kargo-shared-resources",
					SystemResourcesNamespace: "kargo-system-resources",
				},
				externalValidateProjectFn: validation.ValidateProject,
			}
			res, err := svr.UpdateConfigMap(ctx, connect.NewRequest(testCase.req))
			testCase.assertions(t, res, err)
		})
	}
}

func Test_server_updateProjectConfigMap(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-configmap",
			Namespace: testProject.Name,
		},
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPut, "/v1beta1/projects/"+testProject.Name+"/configmaps/"+testConfigMap.Name,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "invalid JSON in request body",
				body:          bytes.NewBufferString("{invalid json"),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "empty data in request body",
				body: mustJSONBody(updateConfigMapRequest{
					Data: map[string]string{},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "ConfigMap does not exist",
				body: mustJSONBody(updateConfigMapRequest{
					Data: map[string]string{"key": "value"},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "updates project ConfigMap",
				body: mustJSONBody(updateConfigMapRequest{
					Description: "updated description",
					Data: map[string]string{
						"key1": "new-value1",
						"key3": "value3",
					},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testConfigMap),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ConfigMap in the response
					resCM := &corev1.ConfigMap{}
					err := json.Unmarshal(w.Body.Bytes(), resCM)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, resCM.Namespace)
					require.Equal(t, testConfigMap.Name, resCM.Name)

					// Verify the ConfigMap was updated in the cluster
					cm := &corev1.ConfigMap{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resCM),
						cm,
					)
					require.NoError(t, err)
					require.Equal(t, "updated description", cm.Annotations[kargoapi.AnnotationKeyDescription])
					// key1 should be updated, key2 removed, key3 added
					require.Equal(t, "new-value1", cm.Data["key1"])
					require.NotContains(t, cm.Data, "key2")
					require.Equal(t, "value3", cm.Data["key3"])
				},
			},
		},
	)
}

func Test_server_updateSystemConfigMap(t *testing.T) {
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-configmap",
			Namespace: testSystemResourcesNamespace,
		},
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{SystemResourcesNamespace: testSystemResourcesNamespace},
		http.MethodPut, "/v1beta1/system/configmaps/"+testConfigMap.Name,
		[]restTestCase{
			{
				name: "invalid JSON in request body",
				body: bytes.NewBufferString("{invalid json"),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "empty data in request body",
				body: mustJSONBody(updateConfigMapRequest{
					Data: map[string]string{},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "ConfigMap does not exist",
				body: mustJSONBody(updateConfigMapRequest{
					Data: map[string]string{"key": "value"},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "updates ConfigMap",
				body: mustJSONBody(updateConfigMapRequest{
					Data: map[string]string{
						"key1": "new-value1",
						"key3": "value3",
					},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testConfigMap),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					resCM := &corev1.ConfigMap{}
					err := json.Unmarshal(w.Body.Bytes(), resCM)
					require.NoError(t, err)
					require.Equal(t, testSystemResourcesNamespace, resCM.Namespace)
					require.Equal(t, testConfigMap.Name, resCM.Name)

					// Verify the ConfigMap was updated in the cluster
					cm := &corev1.ConfigMap{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resCM),
						cm,
					)
					require.NoError(t, err)
					// key1 should be updated, key2 removed, key3 added
					require.Equal(t, "new-value1", cm.Data["key1"])
					require.NotContains(t, cm.Data, "key2")
					require.Equal(t, "value3", cm.Data["key3"])
				},
			},
		},
	)
}

func Test_server_updateSharedConfigMap(t *testing.T) {
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-configmap",
			Namespace: testSharedResourcesNamespace,
		},
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{SharedResourcesNamespace: testSharedResourcesNamespace},
		http.MethodPut, "/v1beta1/shared/configmaps/"+testConfigMap.Name,
		[]restTestCase{
			{
				name: "invalid JSON in request body",
				body: bytes.NewBufferString("{invalid json"),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "empty data in request body",
				body: mustJSONBody(updateConfigMapRequest{
					Data: map[string]string{},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "ConfigMap does not exist",
				body: mustJSONBody(updateConfigMapRequest{
					Data: map[string]string{"key": "value"},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "updates ConfigMap",
				body: mustJSONBody(updateConfigMapRequest{
					Data: map[string]string{
						"key1": "new-value1",
						"key3": "value3",
					},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testConfigMap),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					resCM := &corev1.ConfigMap{}
					err := json.Unmarshal(w.Body.Bytes(), resCM)
					require.NoError(t, err)
					require.Equal(t, testSharedResourcesNamespace, resCM.Namespace)
					require.Equal(t, testConfigMap.Name, resCM.Name)

					// Verify the ConfigMap was updated in the cluster
					cm := &corev1.ConfigMap{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resCM),
						cm,
					)
					require.NoError(t, err)
					// key1 should be updated, key2 removed, key3 added
					require.Equal(t, "new-value1", cm.Data["key1"])
					require.NotContains(t, cm.Data, "key2")
					require.Equal(t, "value3", cm.Data["key3"])
				},
			},
		},
	)
}
