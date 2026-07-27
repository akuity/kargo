package server

import (
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

func Test_server_getProjectGenericCredentials(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-credential",
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
			},
		},
		Data: map[string][]byte{
			"secret-key": []byte("secret-value"),
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{SecretManagementEnabled: true},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/generic-credentials/"+testCreds.Name,
		[]restTestCase{
			{
				name:          "secret management disabled",
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
				name:          "credentials do not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not labeled as generic credentials",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					func() *corev1.Secret {
						secret := testCreds.DeepCopy()
						delete(secret.Labels, kargoapi.LabelKeyCredentialType)
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "Secret exists but is labeled as repo credentials",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					func() *corev1.Secret {
						secret := testCreds.DeepCopy()
						secret.Labels[kargoapi.LabelKeyCredentialType] =
							kargoapi.LabelValueCredentialTypeGit
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "gets credentials with redacted data",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testCreds,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Secret in the response
					secret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), secret)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, secret.Namespace)
					require.Equal(t, testCreds.Name, secret.Name)
					// Verify data is redacted
					require.Equal(t, redacted, secret.StringData["secret-key"])
				},
			},
		},
	)
}

func Test_server_getSystemGenericCredentials(t *testing.T) {
	testCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testSystemResourcesNamespace,
			Name:      "fake-credential",
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
			},
		},
		Data: map[string][]byte{
			"secret-key": []byte("secret-value"),
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{
			SecretManagementEnabled:  true,
			SystemResourcesNamespace: testSystemResourcesNamespace,
		},
		http.MethodGet, "/v1beta1/system/generic-credentials/"+testCreds.Name,
		[]restTestCase{
			{
				name: "secret management disabled",
				serverConfig: &config.ServerConfig{
					SecretManagementEnabled:  false,
					SystemResourcesNamespace: testSystemResourcesNamespace,
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name: "credentials do not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not labeled as generic credentials",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					func() *corev1.Secret {
						secret := testCreds.DeepCopy()
						delete(secret.Labels, kargoapi.LabelKeyCredentialType)
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name:          "gets credentials with redacted data",
				clientBuilder: fake.NewClientBuilder().WithObjects(testCreds),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Secret in the response
					secret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), secret)
					require.NoError(t, err)
					require.Equal(t, testSystemResourcesNamespace, secret.Namespace)
					require.Equal(t, testCreds.Name, secret.Name)
					// Verify data is redacted
					require.Equal(t, redacted, secret.StringData["secret-key"])
				},
			},
		},
	)
}

func Test_server_getSharedGenericCredentials(t *testing.T) {
	testCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testSharedResourcesNamespace,
			Name:      "fake-credential",
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGeneric,
			},
		},
		Data: map[string][]byte{
			"secret-key": []byte("secret-value"),
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{
			SecretManagementEnabled:  true,
			SharedResourcesNamespace: testSharedResourcesNamespace,
		},
		http.MethodGet, "/v1beta1/shared/generic-credentials/"+testCreds.Name,
		[]restTestCase{
			{
				name: "secret management disabled",
				serverConfig: &config.ServerConfig{
					SecretManagementEnabled:  false,
					SharedResourcesNamespace: testSharedResourcesNamespace,
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name: "credentials do not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not labeled as generic credentials",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					func() *corev1.Secret {
						secret := testCreds.DeepCopy()
						delete(secret.Labels, kargoapi.LabelKeyCredentialType)
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name:          "gets credentials with redacted data",
				clientBuilder: fake.NewClientBuilder().WithObjects(testCreds),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Secret in the response
					secret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), secret)
					require.NoError(t, err)
					require.Equal(t, testSharedResourcesNamespace, secret.Namespace)
					require.Equal(t, testCreds.Name, secret.Name)
					// Verify data is redacted
					require.Equal(t, redacted, secret.StringData["secret-key"])
				},
			},
		},
	)
}
