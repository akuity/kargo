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

func TestApplyPatchRepoCredentialsRequestToK8sSecret(t *testing.T) {
	baseSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
			},
		},
		Data: map[string][]byte{
			libCreds.FieldRepoURL:  []byte("fake-url"),
			libCreds.FieldUsername: []byte("fake-username"),
			libCreds.FieldPassword: []byte("fake-password"),
		},
	}

	t.Run("patch repoURL", func(t *testing.T) {
		expectedSecret := baseSecret.DeepCopy()
		expectedSecret.Data[libCreds.FieldRepoURL] = []byte("new-fake-url")
		secret := baseSecret.DeepCopy()
		applyPatchRepoCredentialsRequestToK8sSecret(
			secret,
			patchRepoCredentialsRequest{
				RepoURL: "new-fake-url",
			},
		)
		require.Equal(t, expectedSecret, secret)
	})

	t.Run("patch repoURL with pattern", func(t *testing.T) {
		expectedSecret := baseSecret.DeepCopy()
		expectedSecret.Data[libCreds.FieldRepoURL] = []byte("new-fake-url")
		expectedSecret.Data[libCreds.FieldRepoURLIsRegex] = []byte("true")
		secret := baseSecret.DeepCopy()
		applyPatchRepoCredentialsRequestToK8sSecret(
			secret,
			patchRepoCredentialsRequest{
				RepoURL:        "new-fake-url",
				RepoURLIsRegex: true,
			},
		)
		require.Equal(t, expectedSecret, secret)
	})

	t.Run("patch username", func(t *testing.T) {
		expectedSecret := baseSecret.DeepCopy()
		expectedSecret.Data["username"] = []byte("new-fake-username")
		secret := baseSecret.DeepCopy()
		applyPatchRepoCredentialsRequestToK8sSecret(
			secret,
			patchRepoCredentialsRequest{
				Username: "new-fake-username",
			},
		)
		require.Equal(t, expectedSecret, secret)
	})

	t.Run("patch password", func(t *testing.T) {
		expectedSecret := baseSecret.DeepCopy()
		expectedSecret.Data["password"] = []byte("new-fake-password")
		secret := baseSecret.DeepCopy()
		applyPatchRepoCredentialsRequestToK8sSecret(
			secret,
			patchRepoCredentialsRequest{
				Password: "new-fake-password",
			},
		)
		require.Equal(t, expectedSecret, secret)
	})

	t.Run("patch description", func(t *testing.T) {
		expectedSecret := baseSecret.DeepCopy()
		expectedSecret.Annotations = map[string]string{
			kargoapi.AnnotationKeyDescription: "new description",
		}
		secret := baseSecret.DeepCopy()
		desc := "new description"
		applyPatchRepoCredentialsRequestToK8sSecret(
			secret,
			patchRepoCredentialsRequest{
				Description: &desc,
			},
		)
		require.Equal(t, expectedSecret, secret)
	})

	t.Run("patch clears description with empty string", func(t *testing.T) {
		secret := baseSecret.DeepCopy()
		secret.Annotations = map[string]string{
			kargoapi.AnnotationKeyDescription: "old description",
		}
		emptyDesc := ""
		applyPatchRepoCredentialsRequestToK8sSecret(
			secret,
			patchRepoCredentialsRequest{
				Description: &emptyDesc,
			},
		)
		_, hasDesc := secret.Annotations[kargoapi.AnnotationKeyDescription]
		require.False(t, hasDesc)
	})

	t.Run("patch nil description leaves it unchanged", func(t *testing.T) {
		secret := baseSecret.DeepCopy()
		secret.Annotations = map[string]string{
			kargoapi.AnnotationKeyDescription: "existing description",
		}
		applyPatchRepoCredentialsRequestToK8sSecret(
			secret,
			patchRepoCredentialsRequest{
				Description: nil,
				Username:    "new-username",
			},
		)
		require.Equal(t, "existing description", secret.Annotations[kargoapi.AnnotationKeyDescription])
		require.Equal(t, "new-username", string(secret.Data["username"]))
	})
}

func Test_server_patchProjectRepoCredentials(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	const (
		testRepoURL     = "https://github.com/example/repo"
		testNewRepoURL  = "https://github.com/example/new-repo"
		testUsername    = "username"
		testNewUsername = "new-username"
		testPassword    = "password"
	)
	testCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-creds",
			Namespace: testProject.Name,
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
			},
		},
		Data: map[string][]byte{
			libCreds.FieldRepoURL:  []byte(testRepoURL),
			libCreds.FieldUsername: []byte(testUsername),
			libCreds.FieldPassword: []byte(testPassword),
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{SecretManagementEnabled: true},
		http.MethodPatch, "/v1beta1/projects/"+testProject.Name+"/repo-credentials/"+testCreds.Name,
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
				name: "credentials do not exist",
				body: mustJSONBody(patchRepoCredentialsRequest{
					RepoURL: testNewRepoURL,
				}),
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not labeled as credentials",
				body: mustJSONBody(patchRepoCredentialsRequest{
					RepoURL: testNewRepoURL,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testCreds.Name,
							Namespace: testProject.Name,
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "patches credentials",
				body: mustJSONBody(patchRepoCredentialsRequest{
					RepoURL:  testNewRepoURL,
					Username: testNewUsername,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testCreds,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

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
					// Password should be redacted in response
					require.Equal(t, redacted, resSecret.StringData[libCreds.FieldPassword])

					// Verify the Secret was updated in the cluster
					secret := &corev1.Secret{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resSecret),
						secret,
					)
					require.NoError(t, err)
					require.Equal(t, testNewRepoURL, string(secret.Data[libCreds.FieldRepoURL]))
					require.Equal(t, testNewUsername, string(secret.Data[libCreds.FieldUsername]))
					// Password should remain unchanged since we didn't update it
					require.Equal(t, testPassword, string(secret.Data[libCreds.FieldPassword]))
				},
			},
		},
	)
}

func Test_server_patchSharedRepoCredentials(t *testing.T) {
	const (
		testRepoURL     = "https://github.com/example/repo"
		testNewRepoURL  = "https://github.com/example/new-repo"
		testUsername    = "username"
		testNewUsername = "new-username"
		testPassword    = "password"
	)
	testCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-creds",
			Namespace: testSharedResourcesNamespace,
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
			},
		},
		Data: map[string][]byte{
			libCreds.FieldRepoURL:  []byte(testRepoURL),
			libCreds.FieldUsername: []byte(testUsername),
			libCreds.FieldPassword: []byte(testPassword),
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{
			SecretManagementEnabled:  true,
			SharedResourcesNamespace: testSharedResourcesNamespace,
		},
		http.MethodPatch, "/v1beta1/shared/repo-credentials/"+testCreds.Name,
		[]restTestCase{
			{
				name:         "Secret management disabled",
				serverConfig: &config.ServerConfig{SecretManagementEnabled: false},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name:          "invalid JSON in request body",
				body:          bytes.NewBufferString("{invalid json"),
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "credentials do not exist",
				body: mustJSONBody(patchRepoCredentialsRequest{
					RepoURL: testNewRepoURL,
				}),
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not labeled as credentials",
				body: mustJSONBody(patchRepoCredentialsRequest{
					RepoURL: testNewRepoURL,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      testCreds.Name,
							Namespace: testSharedResourcesNamespace,
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "patches credentials",
				body: mustJSONBody(patchRepoCredentialsRequest{
					RepoURL:  testNewRepoURL,
					Username: testNewUsername,
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testCreds,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

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
					// Password should be redacted in response
					require.Equal(t, redacted, resSecret.StringData[libCreds.FieldPassword])

					// Verify the Secret was updated in the cluster
					secret := &corev1.Secret{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resSecret),
						secret,
					)
					require.NoError(t, err)
					require.Equal(t, testNewRepoURL, string(secret.Data[libCreds.FieldRepoURL]))
					require.Equal(t, testNewUsername, string(secret.Data[libCreds.FieldUsername]))
					// Password should remain unchanged since we didn't update it
					require.Equal(t, testPassword, string(secret.Data[libCreds.FieldPassword]))
				},
			},
		},
	)
}
