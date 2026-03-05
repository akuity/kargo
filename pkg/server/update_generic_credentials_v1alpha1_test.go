package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/validation"
)

func TestUpdateGenericCredentials(t *testing.T) {
	ctx := context.Background()

	cl, err := kubernetes.NewClient(
		ctx,
		&rest.Config{},
		kubernetes.ClientOptions{
			SkipAuthorization: true,
			NewInternalClient: func(_ context.Context, _ *rest.Config, s *runtime.Scheme) (client.WithWatch, error) {
				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(
						mustNewObject[corev1.Namespace]("testdata/namespace.yaml"),
						&corev1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Namespace: "kargo-demo",
								Name:      "secret",
								Labels: map[string]string{
									kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
								},
							},
							StringData: map[string]string{
								"TOKEN_1": "foo",
								"TOKEN_2": "baz",
							},
						},
					).
					Build(), nil
			},
		},
	)
	require.NoError(t, err)

	s := &server{
		client:                    cl,
		cfg:                       config.ServerConfig{SecretManagementEnabled: true},
		externalValidateProjectFn: validation.ValidateProject,
	}

	_, err = s.UpdateGenericCredentials(ctx, connect.NewRequest(&svcv1alpha1.UpdateGenericCredentialsRequest{
		Project: "kargo-demo",
		Name:    "secret",
		Data: map[string]string{
			"TOKEN_1": "bar",
		},
	}))
	require.NoError(t, err)

	secret := corev1.Secret{}

	require.NoError(t, s.client.Get(ctx, types.NamespacedName{
		Namespace: "kargo-demo",
		Name:      "secret",
	}, &secret))

	secret1, ok := secret.Data["TOKEN_1"]
	require.True(t, ok)
	require.Equal(t, "bar", string(secret1))

	_, ok = secret.Data["TOKEN_2"]
	require.False(t, ok)
}

func Test_server_updateProjectGenericCredentials(t *testing.T) {
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
		http.MethodPut, "/v1beta1/projects/"+testProject.Name+"/generic-credentials/"+testSecret.Name,
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
				name: "empty data in request body",
				body: mustJSONBody(updateGenericCredentialsRequest{
					Data: map[string]string{},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "Secret does not exist",
				body: mustJSONBody(updateGenericCredentialsRequest{
					Data: map[string]string{"key": "value"},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not labeled as a generic credential",
				body: mustJSONBody(updateGenericCredentialsRequest{
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
				name: "updates project Secret",
				body: mustJSONBody(updateGenericCredentialsRequest{
					Description: "updated description",
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
					require.Equal(t, "updated description", secret.Annotations[kargoapi.AnnotationKeyDescription])
					// key1 should be updated, key2 removed, key3 added
					require.Equal(t, "new-value1", string(secret.Data["key1"]))
					require.NotContains(t, secret.Data, "key2")
					require.Equal(t, "value3", string(secret.Data["key3"]))
				},
			},
		},
	)
}

func Test_server_updateSystemGenericCredentials(t *testing.T) {
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
		http.MethodPut, "/v1beta1/system/generic-credentials/"+testSecret.Name,
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
				name: "empty data in request body",
				body: mustJSONBody(updateGenericCredentialsRequest{
					Data: map[string]string{},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "Secret does not exist",
				body: mustJSONBody(updateGenericCredentialsRequest{
					Data: map[string]string{"key": "value"},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "updates Secret",
				body: mustJSONBody(updateGenericCredentialsRequest{
					Data: map[string]string{
						"key1": "new-value1",
						"key3": "value3",
					},
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

func Test_server_updateSharedGenericCredentials(t *testing.T) {
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
		http.MethodPut, "/v1beta1/shared/generic-credentials/"+testSecret.Name,
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
				name: "empty data in request body",
				body: mustJSONBody(updateGenericCredentialsRequest{
					Data: map[string]string{},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "Secret does not exist",
				body: mustJSONBody(updateGenericCredentialsRequest{
					Data: map[string]string{"key": "value"},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "updates Secret",
				body: mustJSONBody(updateGenericCredentialsRequest{
					Data: map[string]string{
						"key1": "new-value1",
						"key3": "value3",
					},
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
