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
	libCreds "github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
)

func TestCreateRepoCredentials(t *testing.T) {
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

	resp, err := s.CreateRepoCredentials(
		ctx,
		connect.NewRequest(
			&svcv1alpha1.CreateRepoCredentialsRequest{
				Project:     "kargo-demo",
				Name:        "creds",
				Description: "my credentials",
				Type:        "git",
				RepoUrl:     "https://github.com/example/repo",
				Username:    "username",
				Password:    "password",
			},
		),
	)
	require.NoError(t, err)

	creds := resp.Msg.GetCredentials()
	assert.Equal(t, "kargo-demo", creds.Namespace)
	assert.Equal(t, "creds", creds.Name)
	assert.Equal(t, "my credentials", creds.Annotations[kargoapi.AnnotationKeyDescription])
	assert.Equal(t, "https://github.com/example/repo", creds.StringData[libCreds.FieldRepoURL])
	assert.Equal(t, "username", creds.StringData[libCreds.FieldUsername])
	assert.Equal(t, redacted, creds.StringData[libCreds.FieldPassword])

	secret := corev1.Secret{}
	err = cl.Get(
		ctx,
		types.NamespacedName{
			Namespace: "kargo-demo",
			Name:      "creds",
		},
		&secret,
	)
	require.NoError(t, err)

	data := secret.Data
	assert.Equal(t, "kargo-demo", secret.Namespace)
	assert.Equal(t, "creds", secret.Name)
	assert.Equal(t, "my credentials", secret.Annotations[kargoapi.AnnotationKeyDescription])
	assert.Equal(t, "https://github.com/example/repo", string(data[libCreds.FieldRepoURL]))
	assert.Equal(t, "username", string(data[libCreds.FieldUsername]))
	assert.Equal(t, "password", string(data[libCreds.FieldPassword]))
}

func Test_server_createProjectRepoCredentials(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-creds",
			Namespace: testProject.Name,
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
			},
		},
	}
	const (
		testRepoURL  = "https://github.com/example/repo"
		testUsername = "username"
		testPassword = "password"
	)
	testRESTEndpoint(
		t, &config.ServerConfig{SecretManagementEnabled: true},
		http.MethodPost, "/v1beta1/projects/"+testProject.Name+"/repo-credentials",
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
				name: "missing name in request body",
				body: mustJSONBody(createRepoCredentialsRequest{
					Type:     kargoapi.LabelValueCredentialTypeGit,
					RepoURL:  testRepoURL,
					Username: testUsername,
					Password: testPassword,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "missing type in request body",
				body: mustJSONBody(createRepoCredentialsRequest{
					Name:     testCreds.Name,
					RepoURL:  testRepoURL,
					Username: testUsername,
					Password: testPassword,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "invalid type in request body",
				body: mustJSONBody(createRepoCredentialsRequest{
					Name:     testCreds.Name,
					Type:     "invalid",
					RepoURL:  testRepoURL,
					Username: testUsername,
					Password: testPassword,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "Secret already exists",
				body: mustJSONBody(createRepoCredentialsRequest{
					Name:     testCreds.Name,
					Type:     kargoapi.LabelValueCredentialTypeGit,
					RepoURL:  testRepoURL,
					Username: testUsername,
					Password: testPassword,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testCreds,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "creates credentials",
				body: mustJSONBody(createRepoCredentialsRequest{
					Name:     testCreds.Name,
					Type:     kargoapi.LabelValueCredentialTypeGit,
					RepoURL:  testRepoURL,
					Username: testUsername,
					Password: testPassword,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					// Examine the Secret in the response
					resSecret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), resSecret)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, resSecret.Namespace)
					require.Equal(t, testCreds.Name, resSecret.Name)
					require.Equal(
						t,
						map[string]string{
							kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
						},
						resSecret.Labels,
					)
					require.Equal(
						t,
						map[string]string{
							libCreds.FieldRepoURL:  testRepoURL,
							libCreds.FieldUsername: testUsername,
							libCreds.FieldPassword: redacted,
						},
						resSecret.StringData,
					)
					require.Nil(t, resSecret.Data)

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
						map[string]string{
							kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
						},
						secret.Labels,
					)
					require.Equal(
						t,
						map[string][]byte{
							libCreds.FieldRepoURL:  []byte(testRepoURL),
							libCreds.FieldUsername: []byte(testUsername),
							libCreds.FieldPassword: []byte(testPassword),
						},
						secret.Data,
					)
				},
			},
		},
	)
}

func Test_server_createSharedRepoCredentials(t *testing.T) {
	testCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-creds",
			Namespace: testSharedResourcesNamespace,
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
			},
		},
	}
	const (
		testRepoURL  = "https://github.com/example/repo"
		testUsername = "username"
		testPassword = "password"
	)
	testRESTEndpoint(
		t, &config.ServerConfig{
			SecretManagementEnabled:  true,
			SharedResourcesNamespace: testSharedResourcesNamespace,
		},
		http.MethodPost, "/v1beta1/shared/repo-credentials",
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
				body: mustJSONBody(createRepoCredentialsRequest{
					Type:     kargoapi.LabelValueCredentialTypeGit,
					RepoURL:  testRepoURL,
					Username: testUsername,
					Password: testPassword,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "missing type in request body",
				body: mustJSONBody(createRepoCredentialsRequest{
					Name:     testCreds.Name,
					RepoURL:  testRepoURL,
					Username: testUsername,
					Password: testPassword,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "invalid type in request body",
				body: mustJSONBody(createRepoCredentialsRequest{
					Name:     testCreds.Name,
					Type:     "invalid",
					RepoURL:  testRepoURL,
					Username: testUsername,
					Password: testPassword,
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "Secret already exists",
				body: mustJSONBody(createRepoCredentialsRequest{
					Name:     testCreds.Name,
					Type:     kargoapi.LabelValueCredentialTypeGit,
					RepoURL:  testRepoURL,
					Username: testUsername,
					Password: testPassword,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testCreds),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "creates credentials",
				body: mustJSONBody(createRepoCredentialsRequest{
					Name:     testCreds.Name,
					Type:     kargoapi.LabelValueCredentialTypeGit,
					RepoURL:  testRepoURL,
					Username: testUsername,
					Password: testPassword,
				}),
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					// Examine the Secret in the response
					resSecret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), resSecret)
					require.NoError(t, err)
					require.Equal(t, testSharedResourcesNamespace, resSecret.Namespace)
					require.Equal(t, testCreds.Name, resSecret.Name)
					require.Equal(
						t,
						map[string]string{
							kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
						},
						resSecret.Labels,
					)
					require.Equal(
						t,
						map[string]string{
							libCreds.FieldRepoURL:  testRepoURL,
							libCreds.FieldUsername: testUsername,
							libCreds.FieldPassword: redacted,
						},
						resSecret.StringData,
					)
					require.Nil(t, resSecret.Data)

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
						map[string]string{
							kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
						},
						secret.Labels,
					)
					require.Equal(
						t,
						map[string][]byte{
							libCreds.FieldRepoURL:  []byte(testRepoURL),
							libCreds.FieldUsername: []byte(testUsername),
							libCreds.FieldPassword: []byte(testPassword),
						},
						secret.Data,
					)
				},
			},
		},
	)
}
