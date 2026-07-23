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
	libCreds "github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/server/config"
)

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
