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

func TestApplyGenericCredentialsPatchToK8sSecret(t *testing.T) {
	baseSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
			},
		},
		Data: map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
		},
	}

	t.Run("merge new data", func(t *testing.T) {
		secret := baseSecret.DeepCopy()
		applyGenericCredentialsPatchToK8sSecret(secret, patchGenericCredentialsRequest{
			Data: map[string]string{
				"key3": "value3",
			},
		})
		require.Equal(t, "value1", string(secret.Data["key1"]))
		require.Equal(t, "value2", string(secret.Data["key2"]))
		require.Equal(t, "value3", string(secret.Data["key3"]))
	})

	t.Run("update existing key", func(t *testing.T) {
		secret := baseSecret.DeepCopy()
		applyGenericCredentialsPatchToK8sSecret(secret, patchGenericCredentialsRequest{
			Data: map[string]string{
				"key1": "new-value1",
			},
		})
		require.Equal(t, "new-value1", string(secret.Data["key1"]))
		require.Equal(t, "value2", string(secret.Data["key2"]))
	})

	t.Run("remove keys", func(t *testing.T) {
		secret := baseSecret.DeepCopy()
		applyGenericCredentialsPatchToK8sSecret(secret, patchGenericCredentialsRequest{
			RemoveKeys: []string{"key1"},
		})
		require.NotContains(t, secret.Data, "key1")
		require.Equal(t, "value2", string(secret.Data["key2"]))
	})

	t.Run("add and remove keys simultaneously", func(t *testing.T) {
		secret := baseSecret.DeepCopy()
		applyGenericCredentialsPatchToK8sSecret(secret, patchGenericCredentialsRequest{
			Data: map[string]string{
				"key3": "value3",
			},
			RemoveKeys: []string{"key1"},
		})
		require.NotContains(t, secret.Data, "key1")
		require.Equal(t, "value2", string(secret.Data["key2"]))
		require.Equal(t, "value3", string(secret.Data["key3"]))
	})

	t.Run("set description", func(t *testing.T) {
		secret := baseSecret.DeepCopy()
		desc := "new description"
		applyGenericCredentialsPatchToK8sSecret(secret, patchGenericCredentialsRequest{
			Description: &desc,
		})
		require.Equal(t, "new description", secret.Annotations[kargoapi.AnnotationKeyDescription])
	})

	t.Run("clear description", func(t *testing.T) {
		secret := baseSecret.DeepCopy()
		secret.Annotations = map[string]string{
			kargoapi.AnnotationKeyDescription: "old description",
		}
		emptyDesc := ""
		applyGenericCredentialsPatchToK8sSecret(secret, patchGenericCredentialsRequest{
			Description: &emptyDesc,
		})
		require.NotContains(t, secret.Annotations, kargoapi.AnnotationKeyDescription)
	})

	t.Run("nil description leaves it unchanged", func(t *testing.T) {
		secret := baseSecret.DeepCopy()
		secret.Annotations = map[string]string{
			kargoapi.AnnotationKeyDescription: "existing description",
		}
		applyGenericCredentialsPatchToK8sSecret(secret, patchGenericCredentialsRequest{
			Data: map[string]string{
				"key3": "value3",
			},
		})
		require.Equal(t, "existing description", secret.Annotations[kargoapi.AnnotationKeyDescription])
	})
}

func Test_server_patchProjectGenericCredentials(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-secret",
			Namespace: testProject.Name,
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
			},
		},
		Data: map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{SecretManagementEnabled: true},
		http.MethodPatch, "/v1beta1/projects/"+testProject.Name+"/generic-credentials/"+testSecret.Name,
		[]restTestCase{
			{
				name:          "Secret management disabled",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				serverConfig:  &config.ServerConfig{SecretManagementEnabled: false},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
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
				name: "Secret does not exist",
				body: mustJSONBody(patchGenericCredentialsRequest{
					Data: map[string]string{"key": "value"},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not labeled as a generic credential",
				body: mustJSONBody(patchGenericCredentialsRequest{
					Data: map[string]string{"key": "value"},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					func() *corev1.Secret {
						s := testSecret.DeepCopy()
						s.Labels[kargoapi.LabelKeyCredentialType] = kargoapi.LabelValueCredentialTypeGit
						return s
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "patch would result in empty secret",
				body: mustJSONBody(patchGenericCredentialsRequest{
					RemoveKeys: []string{"key1", "key2"},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testSecret),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "patches project Secret - add and update keys",
				body: mustJSONBody(patchGenericCredentialsRequest{
					Data: map[string]string{
						"key1": "new-value1",
						"key3": "value3",
					},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testSecret),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Secret in the response
					resSecret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), resSecret)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, resSecret.Namespace)
					require.Equal(t, testSecret.Name, resSecret.Name)

					// Verify the Secret was updated in the cluster
					secret := &corev1.Secret{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resSecret),
						secret,
					)
					require.NoError(t, err)
					// key1 should be updated, key2 preserved, key3 added
					require.Equal(t, "new-value1", string(secret.Data["key1"]))
					require.Equal(t, "value2", string(secret.Data["key2"]))
					require.Equal(t, "value3", string(secret.Data["key3"]))
				},
			},
			{
				name: "patches project Secret - remove key",
				body: mustJSONBody(patchGenericCredentialsRequest{
					RemoveKeys: []string{"key1"},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testSecret),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify the Secret was updated in the cluster
					resSecret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), resSecret)
					require.NoError(t, err)

					secret := &corev1.Secret{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resSecret),
						secret,
					)
					require.NoError(t, err)
					// key1 should be removed, key2 preserved
					require.NotContains(t, secret.Data, "key1")
					require.Equal(t, "value2", string(secret.Data["key2"]))
				},
			},
			{
				name: "patches project Secret - update description",
				body: func() *bytes.Buffer {
					desc := "new description"
					data, _ := json.Marshal(patchGenericCredentialsRequest{
						Description: &desc,
					})
					return bytes.NewBuffer(data)
				}(),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testSecret),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					resSecret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), resSecret)
					require.NoError(t, err)

					secret := &corev1.Secret{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resSecret),
						secret,
					)
					require.NoError(t, err)
					require.Equal(t, "new description", secret.Annotations[kargoapi.AnnotationKeyDescription])
				},
			},
		},
	)
}

func Test_server_patchSystemGenericCredentials(t *testing.T) {
	testSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-secret",
			Namespace: testSystemResourcesNamespace,
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
			},
		},
		Data: map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{
			SecretManagementEnabled:  true,
			SystemResourcesNamespace: testSystemResourcesNamespace,
		},
		http.MethodPatch, "/v1beta1/system/generic-credentials/"+testSecret.Name,
		[]restTestCase{
			{
				name:         "Secret management disabled",
				serverConfig: &config.ServerConfig{SecretManagementEnabled: false},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
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
				name: "Secret does not exist",
				body: mustJSONBody(patchGenericCredentialsRequest{
					Data: map[string]string{"key": "value"},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not labeled as a generic credential",
				body: mustJSONBody(patchGenericCredentialsRequest{
					Data: map[string]string{"key": "value"},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					func() *corev1.Secret {
						s := testSecret.DeepCopy()
						s.Labels[kargoapi.LabelKeyCredentialType] = kargoapi.LabelValueCredentialTypeGit
						return s
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "patch would result in empty secret",
				body: mustJSONBody(patchGenericCredentialsRequest{
					RemoveKeys: []string{"key1", "key2"},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testSecret),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "patches system Secret",
				body: mustJSONBody(patchGenericCredentialsRequest{
					Data: map[string]string{
						"key1": "new-value1",
						"key3": "value3",
					},
					RemoveKeys: []string{"key2"},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testSecret),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					resSecret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), resSecret)
					require.NoError(t, err)
					require.Equal(t, testSystemResourcesNamespace, resSecret.Namespace)
					require.Equal(t, testSecret.Name, resSecret.Name)

					// Verify the Secret was updated in the cluster
					secret := &corev1.Secret{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resSecret),
						secret,
					)
					require.NoError(t, err)
					// key1 should be updated, key2 removed, key3 added
					require.Equal(t, "new-value1", string(secret.Data["key1"]))
					require.NotContains(t, secret.Data, "key2")
					require.Equal(t, "value3", string(secret.Data["key3"]))
				},
			},
		},
	)
}

func Test_server_patchSharedGenericCredentials(t *testing.T) {
	testSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-secret",
			Namespace: testSharedResourcesNamespace,
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
			},
		},
		Data: map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{
			SecretManagementEnabled:  true,
			SharedResourcesNamespace: testSharedResourcesNamespace,
		},
		http.MethodPatch, "/v1beta1/shared/generic-credentials/"+testSecret.Name,
		[]restTestCase{
			{
				name:         "Secret management disabled",
				serverConfig: &config.ServerConfig{SecretManagementEnabled: false},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
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
				name: "Secret does not exist",
				body: mustJSONBody(patchGenericCredentialsRequest{
					Data: map[string]string{"key": "value"},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not labeled as a generic credential",
				body: mustJSONBody(patchGenericCredentialsRequest{
					Data: map[string]string{"key": "value"},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					func() *corev1.Secret {
						s := testSecret.DeepCopy()
						s.Labels[kargoapi.LabelKeyCredentialType] = kargoapi.LabelValueCredentialTypeGit
						return s
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "patch would result in empty secret",
				body: mustJSONBody(patchGenericCredentialsRequest{
					RemoveKeys: []string{"key1", "key2"},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testSecret),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "patches shared Secret",
				body: mustJSONBody(patchGenericCredentialsRequest{
					Data: map[string]string{
						"key1": "new-value1",
						"key3": "value3",
					},
					RemoveKeys: []string{"key2"},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testSecret),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					resSecret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), resSecret)
					require.NoError(t, err)
					require.Equal(t, testSharedResourcesNamespace, resSecret.Namespace)
					require.Equal(t, testSecret.Name, resSecret.Name)

					// Verify the Secret was updated in the cluster
					secret := &corev1.Secret{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resSecret),
						secret,
					)
					require.NoError(t, err)
					// key1 should be updated, key2 removed, key3 added
					require.Equal(t, "new-value1", string(secret.Data["key1"]))
					require.NotContains(t, secret.Data, "key2")
					require.Equal(t, "value3", string(secret.Data["key3"]))
				},
			},
		},
	)
}
