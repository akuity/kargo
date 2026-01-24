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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
)

func TestCreateGenericCredentials(t *testing.T) {
	ctx := context.Background()

	cl, err := kubernetes.NewClient(
		ctx,
		&rest.Config{},
		kubernetes.ClientOptions{
			SkipAuthorization: true,
			NewInternalClient: func(_ context.Context, _ *rest.Config, s *runtime.Scheme) (client.WithWatch, error) {
				return fake.NewClientBuilder().
					WithScheme(s).
					WithObjects(mustNewObject[corev1.Namespace]("testdata/namespace.yaml")).
					Build(), nil
			},
		},
	)
	require.NoError(t, err)

	s := &server{
		client: cl,
		cfg:    config.ServerConfig{SecretManagementEnabled: true},
		externalValidateProjectFn: func(context.Context, client.Client, string) error {
			return nil
		},
	}

	resp, err := s.CreateGenericCredentials(ctx, connect.NewRequest(&svcv1alpha1.CreateGenericCredentialsRequest{
		Project:     "kargo-demo",
		Name:        "secret",
		Description: "my secret",
		Data: map[string]string{
			"TOKEN_1": "foo",
			"TOKEN_2": "bar",
		},
	}))
	require.NoError(t, err)

	genCreds := resp.Msg.GetCredentials()
	assert.Equal(t, "kargo-demo", genCreds.Namespace)
	assert.Equal(t, "secret", genCreds.Name)
	assert.Equal(t, "my secret", genCreds.Annotations[kargoapi.AnnotationKeyDescription])
	assert.Equal(t, redacted, genCreds.StringData["TOKEN_1"])
	assert.Equal(t, redacted, genCreds.StringData["TOKEN_2"])

	secret := corev1.Secret{}
	err = cl.Get(ctx, types.NamespacedName{
		Namespace: "kargo-demo",
		Name:      "secret",
	},
		&secret,
	)
	require.NoError(t, err)

	data := secret.Data
	assert.Equal(t, "kargo-demo", secret.Namespace)
	assert.Equal(t, "secret", secret.Name)
	assert.Equal(t, "my secret", secret.Annotations[kargoapi.AnnotationKeyDescription])
	assert.Equal(t, "foo", string(data["TOKEN_1"]))
	assert.Equal(t, "bar", string(data["TOKEN_2"]))
}

func Test_server_createProjectGenericCredentials(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-secret",
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
			},
		},
	}
	const testDescription = "fake description"
	testData := map[string]string{"foo": "bar", "bat": "baz"}
	testRESTEndpoint(
		t, &config.ServerConfig{SecretManagementEnabled: true},
		http.MethodPost, "/v1beta1/projects/"+testProject.Name+"/generic-credentials",
		[]restTestCase{
			{
				name:          "Secret management disabled",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				body: mustJSONBody(createGenericCredentialsRequest{
					Name: testSecret.Name,
					Data: testData,
				}),
				serverConfig: &config.ServerConfig{SecretManagementEnabled: false},
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
				name: "missing name in request body",
				body: mustJSONBody(createGenericCredentialsRequest{
					Data: testData,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "empty data in request body",
				body: mustJSONBody(createGenericCredentialsRequest{
					Name: testSecret.Name,
					Data: map[string]string{},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "Secret already exists",
				body: mustJSONBody(createGenericCredentialsRequest{
					Name: testSecret.Name,
					Data: testData,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testSecret,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "creates Secret",
				body: mustJSONBody(createGenericCredentialsRequest{
					Name:        testSecret.Name,
					Description: testDescription,
					Data:        testData,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					// Examine the Secret in the response
					resSecret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), resSecret)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, resSecret.Namespace)
					require.Equal(t, testSecret.Name, resSecret.Name)
					require.Equal(
						t,
						testDescription,
						resSecret.Annotations[kargoapi.AnnotationKeyDescription],
					)
					require.Equal(
						t,
						map[string]string{"foo": redacted, "bat": redacted},
						resSecret.StringData,
					)

					// Verify the Secret was created in the cluster
					secret := &corev1.Secret{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resSecret),
						secret,
					)
					require.NoError(t, err)
					require.Equal(
						t,
						testDescription,
						secret.Annotations[kargoapi.AnnotationKeyDescription],
					)
					require.Equal(
						t,
						map[string][]byte{"foo": []byte("bar"), "bat": []byte("baz")},
						secret.Data,
					)
				},
			},
		},
	)
}

func Test_server_createSystemGenericCredentials(t *testing.T) {
	testSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testSystemResourcesNamespace,
			Name:      "fake-secret",
		},
	}
	const testDescription = "fake description"
	testData := map[string]string{"foo": "bar", "bat": "baz"}
	testRESTEndpoint(
		t, &config.ServerConfig{
			SecretManagementEnabled:  true,
			SystemResourcesNamespace: testSystemResourcesNamespace,
		},
		http.MethodPost, "/v1beta1/system/generic-credentials",
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
				name: "missing name in request body",
				body: mustJSONBody(createGenericCredentialsRequest{Data: testData}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "missing data in request body",
				body: mustJSONBody(createGenericCredentialsRequest{Name: testSecret.Name}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "empty data in request body",
				body: mustJSONBody(createGenericCredentialsRequest{
					Name: testSecret.Name,
					Data: map[string]string{},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "Secret already exists",
				body: mustJSONBody(createGenericCredentialsRequest{
					Name: testSecret.Name,
					Data: testData,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testSecret),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "creates Secret",
				body: mustJSONBody(createGenericCredentialsRequest{
					Name:        testSecret.Name,
					Description: testDescription,
					Data:        testData,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					// Examine the Secret in the response
					resSecret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), resSecret)
					require.NoError(t, err)
					require.Equal(t, testSystemResourcesNamespace, resSecret.Namespace)
					require.Equal(t, testSecret.Name, resSecret.Name)
					require.Equal(
						t,
						testDescription,
						resSecret.Annotations[kargoapi.AnnotationKeyDescription],
					)
					require.Equal(
						t,
						map[string]string{"foo": redacted, "bat": redacted},
						resSecret.StringData,
					)

					// Verify the Secret was created in the cluster
					secret := &corev1.Secret{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resSecret),
						secret,
					)
					require.NoError(t, err)
					require.Equal(
						t,
						testDescription,
						secret.Annotations[kargoapi.AnnotationKeyDescription],
					)
					require.Equal(
						t,
						map[string][]byte{"foo": []byte("bar"), "bat": []byte("baz")},
						secret.Data,
					)
				},
			},
		},
	)
}

func Test_server_createSharedGenericCredentials(t *testing.T) {
	testSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testSharedResourcesNamespace,
			Name:      "fake-secret",
		},
	}
	const testDescription = "fake description"
	testData := map[string]string{"foo": "bar", "bat": "baz"}
	testRESTEndpoint(
		t, &config.ServerConfig{
			SecretManagementEnabled:  true,
			SharedResourcesNamespace: testSharedResourcesNamespace,
		},
		http.MethodPost, "/v1beta1/shared/generic-credentials",
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
				name: "missing name in request body",
				body: mustJSONBody(createGenericCredentialsRequest{Data: testData}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "missing data in request body",
				body: mustJSONBody(createGenericCredentialsRequest{Name: testSecret.Name}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "empty data in request body",
				body: mustJSONBody(createGenericCredentialsRequest{
					Name: testSecret.Name,
					Data: map[string]string{},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "Secret already exists",
				body: mustJSONBody(createGenericCredentialsRequest{
					Name: testSecret.Name,
					Data: testData,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testSecret),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "creates Secret",
				body: mustJSONBody(createGenericCredentialsRequest{
					Name:        testSecret.Name,
					Description: testDescription,
					Data:        testData,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					// Examine the Secret in the response
					resSecret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), resSecret)
					require.NoError(t, err)
					require.Equal(t, testSharedResourcesNamespace, resSecret.Namespace)
					require.Equal(t, testSecret.Name, resSecret.Name)
					require.Equal(
						t,
						testDescription,
						resSecret.Annotations[kargoapi.AnnotationKeyDescription],
					)
					require.Equal(
						t,
						map[string]string{"foo": redacted, "bat": redacted},
						resSecret.StringData,
					)

					// Verify the Secret was created in the cluster
					secret := &corev1.Secret{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resSecret),
						secret,
					)
					require.NoError(t, err)
					require.Equal(
						t,
						testDescription,
						secret.Annotations[kargoapi.AnnotationKeyDescription],
					)
					require.Equal(
						t,
						map[string][]byte{"foo": []byte("bar"), "bat": []byte("baz")},
						secret.Data,
					)
				},
			},
		},
	)
}
