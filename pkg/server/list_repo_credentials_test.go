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

func Test_server_listProjectRepoCredentials(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/repo-credentials",
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "no credentials exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &corev1.SecretList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists credentials",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "creds-1",
							Labels: map[string]string{
								kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
							},
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "creds-2",
							Labels: map[string]string{
								kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
							},
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Secrets in the response
					secrets := &corev1.SecretList{}
					err := json.Unmarshal(w.Body.Bytes(), secrets)
					require.NoError(t, err)
					require.Len(t, secrets.Items, 2)
				},
			},
		},
	)
}

func Test_server_listSharedRepoCredentials(t *testing.T) {
	testRESTEndpoint(
		t, &config.ServerConfig{
			SharedResourcesNamespace: testSharedResourcesNamespace,
		},
		http.MethodGet, "/v1beta1/shared/repo-credentials",
		[]restTestCase{
			{
				name:          "no credentials exist",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &corev1.SecretList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists credentials",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testSharedResourcesNamespace,
							Name:      "creds-1",
							Labels: map[string]string{
								kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
							},
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testSharedResourcesNamespace,
							Name:      "creds-2",
							Labels: map[string]string{
								kargoapi.LabelKeyCredentialType: kargoapi.LabelValueCredentialTypeGit,
							},
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Secrets in the response
					secrets := &corev1.SecretList{}
					err := json.Unmarshal(w.Body.Bytes(), secrets)
					require.NoError(t, err)
					require.Len(t, secrets.Items, 2)
				},
			},
		},
	)
}
