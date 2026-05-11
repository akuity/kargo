package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_deleteProjectRepoCredentials(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-credentials",
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
			},
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{SecretManagementEnabled: true},
		http.MethodDelete, "/v1beta1/projects/"+testProject.Name+"/repo-credentials/"+testCreds.Name, []restTestCase{
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
				name:          "Credentials do not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not labeled as repo credentials",
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
				name: "Secret exists but is labeled with wrong credential type",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					func() *corev1.Secret {
						secret := testCreds.DeepCopy()
						secret.Labels[kargoapi.LabelKeyCredentialType] = "some-other-type"
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "deletes credentials",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testCreds,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					// Verify the Secret was deleted from the cluster
					secret := &corev1.Secret{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testCreds),
						secret,
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))
				},
			},
		},
	)
}

func Test_server_deleteSharedRepoCredentials(t *testing.T) {
	testCreds := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testSharedResourcesNamespace,
			Name:      "fake-credentials",
			Labels: map[string]string{
				kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
			},
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{
			SecretManagementEnabled:  true,
			SharedResourcesNamespace: testSharedResourcesNamespace,
		},
		http.MethodDelete, "/v1beta1/shared/repo-credentials/"+testCreds.Name, []restTestCase{
			{
				name:         "Secret management disabled",
				serverConfig: &config.ServerConfig{SecretManagementEnabled: false},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotImplemented, w.Code)
				},
			},
			{
				name:          "Credentials do not exist",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not labeled as repo credentials",
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
				name: "Secret exists but is labeled with wrong credential type",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					func() *corev1.Secret {
						secret := testCreds.DeepCopy()
						secret.Labels[kargoapi.LabelKeyCredentialType] = "some-other-type"
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name:          "deletes credentials",
				clientBuilder: fake.NewClientBuilder().WithObjects(testCreds),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					// Verify the Secret was deleted from the cluster
					secret := &corev1.Secret{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testCreds),
						secret,
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))
				},
			},
		},
	)
}
