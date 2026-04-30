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

func TestApplyConfigMapPatchToK8sConfigMap(t *testing.T) {
	baseConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				kargoapi.AnnotationKeyDescription: "old description",
			},
		},
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
	}

	t.Run("patch adds new keys", func(t *testing.T) {
		cm := baseConfigMap.DeepCopy()
		applyConfigMapPatchToK8sConfigMap(
			cm,
			patchConfigMapRequest{
				Data: map[string]string{
					"key4": "value4",
				},
			},
		)
		require.Equal(t, "value1", cm.Data["key1"])
		require.Equal(t, "value2", cm.Data["key2"])
		require.Equal(t, "value3", cm.Data["key3"])
		require.Equal(t, "value4", cm.Data["key4"])
	})

	t.Run("patch updates existing keys", func(t *testing.T) {
		cm := baseConfigMap.DeepCopy()
		applyConfigMapPatchToK8sConfigMap(
			cm,
			patchConfigMapRequest{
				Data: map[string]string{
					"key1": "new-value1",
				},
			},
		)
		require.Equal(t, "new-value1", cm.Data["key1"])
		require.Equal(t, "value2", cm.Data["key2"])
		require.Equal(t, "value3", cm.Data["key3"])
	})

	t.Run("patch removes keys", func(t *testing.T) {
		cm := baseConfigMap.DeepCopy()
		applyConfigMapPatchToK8sConfigMap(
			cm,
			patchConfigMapRequest{
				RemoveKeys: []string{"key2", "key3"},
			},
		)
		require.Equal(t, "value1", cm.Data["key1"])
		require.NotContains(t, cm.Data, "key2")
		require.NotContains(t, cm.Data, "key3")
	})

	t.Run("patch updates description", func(t *testing.T) {
		cm := baseConfigMap.DeepCopy()
		newDesc := "new description"
		applyConfigMapPatchToK8sConfigMap(
			cm,
			patchConfigMapRequest{
				Description: &newDesc,
			},
		)
		require.Equal(t, "new description", cm.Annotations[kargoapi.AnnotationKeyDescription])
	})

	t.Run("patch clears description with empty string", func(t *testing.T) {
		cm := baseConfigMap.DeepCopy()
		emptyDesc := ""
		applyConfigMapPatchToK8sConfigMap(
			cm,
			patchConfigMapRequest{
				Description: &emptyDesc,
			},
		)
		_, hasDesc := cm.Annotations[kargoapi.AnnotationKeyDescription]
		require.False(t, hasDesc)
	})

	t.Run("patch nil description leaves it unchanged", func(t *testing.T) {
		cm := baseConfigMap.DeepCopy()
		applyConfigMapPatchToK8sConfigMap(
			cm,
			patchConfigMapRequest{
				Description: nil,
				Data: map[string]string{
					"key4": "value4",
				},
			},
		)
		require.Equal(t, "old description", cm.Annotations[kargoapi.AnnotationKeyDescription])
	})

	t.Run("patch combined operations", func(t *testing.T) {
		cm := baseConfigMap.DeepCopy()
		newDesc := "updated description"
		applyConfigMapPatchToK8sConfigMap(
			cm,
			patchConfigMapRequest{
				Description: &newDesc,
				Data: map[string]string{
					"key1": "updated-value1",
					"key4": "value4",
				},
				RemoveKeys: []string{"key2"},
			},
		)
		require.Equal(t, "updated description", cm.Annotations[kargoapi.AnnotationKeyDescription])
		require.Equal(t, "updated-value1", cm.Data["key1"])
		require.NotContains(t, cm.Data, "key2")
		require.Equal(t, "value3", cm.Data["key3"])
		require.Equal(t, "value4", cm.Data["key4"])
	})
}

func TestValidateConfigMapNotEmpty(t *testing.T) {
	t.Run("non-empty ConfigMap is valid", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			Data: map[string]string{"key": "value"},
		}
		err := validateConfigMapNotEmpty(cm)
		require.NoError(t, err)
	})

	t.Run("empty ConfigMap is invalid", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			Data: map[string]string{},
		}
		err := validateConfigMapNotEmpty(cm)
		require.ErrorIs(t, err, errEmptyConfigMap)
	})

	t.Run("nil data is invalid", func(t *testing.T) {
		cm := &corev1.ConfigMap{
			Data: nil,
		}
		err := validateConfigMapNotEmpty(cm)
		require.ErrorIs(t, err, errEmptyConfigMap)
	})
}

func Test_server_patchProjectConfigMap(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-configmap",
			Namespace: testProject.Name,
			Annotations: map[string]string{
				kargoapi.AnnotationKeyDescription: "original description",
			},
		},
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPatch, "/v1beta1/projects/"+testProject.Name+"/configmaps/"+testConfigMap.Name,
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
				name: "ConfigMap does not exist",
				body: func() *bytes.Buffer {
					data, _ := json.Marshal(patchConfigMapRequest{
						Data: map[string]string{"key": "value"},
					})
					return bytes.NewBuffer(data)
				}(),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "patch would result in empty ConfigMap",
				body: func() *bytes.Buffer {
					data, _ := json.Marshal(patchConfigMapRequest{
						RemoveKeys: []string{"key1", "key2"},
					})
					return bytes.NewBuffer(data)
				}(),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testConfigMap),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "patches project ConfigMap - add and update keys",
				body: func() *bytes.Buffer {
					data, _ := json.Marshal(patchConfigMapRequest{
						Data: map[string]string{
							"key1": "new-value1",
							"key3": "value3",
						},
					})
					return bytes.NewBuffer(data)
				}(),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testConfigMap),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

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
					// key1 updated, key2 unchanged, key3 added
					require.Equal(t, "new-value1", cm.Data["key1"])
					require.Equal(t, "value2", cm.Data["key2"])
					require.Equal(t, "value3", cm.Data["key3"])
				},
			},
			{
				name: "patches project ConfigMap - remove key",
				body: func() *bytes.Buffer {
					data, _ := json.Marshal(patchConfigMapRequest{
						RemoveKeys: []string{"key1"},
					})
					return bytes.NewBuffer(data)
				}(),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testConfigMap),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					resCM := &corev1.ConfigMap{}
					err := json.Unmarshal(w.Body.Bytes(), resCM)
					require.NoError(t, err)

					cm := &corev1.ConfigMap{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resCM),
						cm,
					)
					require.NoError(t, err)
					require.NotContains(t, cm.Data, "key1")
					require.Equal(t, "value2", cm.Data["key2"])
				},
			},
			{
				name: "patches project ConfigMap - update description",
				body: func() *bytes.Buffer {
					desc := "new description"
					data, _ := json.Marshal(patchConfigMapRequest{
						Description: &desc,
					})
					return bytes.NewBuffer(data)
				}(),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testConfigMap),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					resCM := &corev1.ConfigMap{}
					err := json.Unmarshal(w.Body.Bytes(), resCM)
					require.NoError(t, err)

					cm := &corev1.ConfigMap{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resCM),
						cm,
					)
					require.NoError(t, err)
					require.Equal(t, "new description", cm.Annotations[kargoapi.AnnotationKeyDescription])
				},
			},
		},
	)
}

func Test_server_patchSystemConfigMap(t *testing.T) {
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
		http.MethodPatch, "/v1beta1/system/configmaps/"+testConfigMap.Name,
		[]restTestCase{
			{
				name: "invalid JSON in request body",
				body: bytes.NewBufferString("{invalid json"),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "ConfigMap does not exist",
				body: func() *bytes.Buffer {
					data, _ := json.Marshal(patchConfigMapRequest{
						Data: map[string]string{"key": "value"},
					})
					return bytes.NewBuffer(data)
				}(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "patch would result in empty ConfigMap",
				body: func() *bytes.Buffer {
					data, _ := json.Marshal(patchConfigMapRequest{
						RemoveKeys: []string{"key1", "key2"},
					})
					return bytes.NewBuffer(data)
				}(),
				clientBuilder: fake.NewClientBuilder().WithObjects(testConfigMap),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "patches ConfigMap",
				body: func() *bytes.Buffer {
					data, _ := json.Marshal(patchConfigMapRequest{
						Data: map[string]string{
							"key1": "new-value1",
							"key3": "value3",
						},
					})
					return bytes.NewBuffer(data)
				}(),
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
					// key1 updated, key2 unchanged, key3 added
					require.Equal(t, "new-value1", cm.Data["key1"])
					require.Equal(t, "value2", cm.Data["key2"])
					require.Equal(t, "value3", cm.Data["key3"])
				},
			},
		},
	)
}

func Test_server_patchSharedConfigMap(t *testing.T) {
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
		http.MethodPatch, "/v1beta1/shared/configmaps/"+testConfigMap.Name,
		[]restTestCase{
			{
				name: "invalid JSON in request body",
				body: bytes.NewBufferString("{invalid json"),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "ConfigMap does not exist",
				body: func() *bytes.Buffer {
					data, _ := json.Marshal(patchConfigMapRequest{
						Data: map[string]string{"key": "value"},
					})
					return bytes.NewBuffer(data)
				}(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "patch would result in empty ConfigMap",
				body: func() *bytes.Buffer {
					data, _ := json.Marshal(patchConfigMapRequest{
						RemoveKeys: []string{"key1", "key2"},
					})
					return bytes.NewBuffer(data)
				}(),
				clientBuilder: fake.NewClientBuilder().WithObjects(testConfigMap),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "patches ConfigMap",
				body: func() *bytes.Buffer {
					data, _ := json.Marshal(patchConfigMapRequest{
						Data: map[string]string{
							"key1": "new-value1",
							"key3": "value3",
						},
					})
					return bytes.NewBuffer(data)
				}(),
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
					// key1 updated, key2 unchanged, key3 added
					require.Equal(t, "new-value1", cm.Data["key1"])
					require.Equal(t, "value2", cm.Data["key2"])
					require.Equal(t, "value3", cm.Data["key3"])
				},
			},
		},
	)
}
