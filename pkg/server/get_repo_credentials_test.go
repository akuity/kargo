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
	libCreds "github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_getProjectRepoCredentials(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-credential",
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: "git",
			},
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/repo-credentials/"+testCreds.Name,
		[]restTestCase{
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
				name: "Secret exists but is not labeled as credentials",
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
				name: "Secret exists but is labeled as generic credentials",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					func() *corev1.Secret {
						secret := testCreds.DeepCopy()
						secret.Labels[kargoapi.LabelKeyCredentialType] =
							kargoapi.LabelValueCredentialTypeGeneric
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "gets credentials",
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
				},
			},
		},
	)
}

func Test_server_getSharedRepoCredentials(t *testing.T) {
	testCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testSharedResourcesNamespace,
			Name:      "fake-credential",
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: "git",
			},
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{SharedResourcesNamespace: testSharedResourcesNamespace},
		http.MethodGet, "/v1beta1/shared/repo-credentials/"+testCreds.Name,
		[]restTestCase{
			{
				name:          "credentials do not exist",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not labeled as credentials",
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
				name: "Secret exists but is labeled as generic credentials",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					func() *corev1.Secret {
						secret := testCreds.DeepCopy()
						secret.Labels[kargoapi.LabelKeyCredentialType] =
							kargoapi.LabelValueCredentialTypeGeneric
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "gets credentials",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testCreds,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Secret in the response
					secret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), secret)
					require.NoError(t, err)
					require.Equal(t, testSharedResourcesNamespace, secret.Namespace)
					require.Equal(t, testCreds.Name, secret.Name)
				},
			},
		},
	)
}

func TestSanitizeCredentialSecret(t *testing.T) {
	creds := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"last-applied-configuration": "fake-configuration",
			},
		},
		Data: map[string][]byte{
			libCreds.FieldRepoURL:  []byte("fake-url"),
			libCreds.FieldUsername: []byte("fake-username"),
			libCreds.FieldPassword: []byte("fake-password"),
			"random-key":           []byte("random-value"),
		},
	}
	sanitizedCreds := sanitizeCredentialSecret(creds)
	require.Equal(
		t,
		map[string]string{
			"last-applied-configuration": redacted,
		},
		sanitizedCreds.Annotations,
	)
	require.Equal(
		t,
		map[string]string{
			libCreds.FieldRepoURL:  "fake-url",
			libCreds.FieldUsername: "fake-username",
			libCreds.FieldPassword: redacted,
			"random-key":           redacted,
		},
		sanitizedCreds.StringData,
	)
}
