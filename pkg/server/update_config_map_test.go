package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

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
