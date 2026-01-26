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

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_getProjectAPIToken(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testToken := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-token",
			Labels: map[string]string{
				rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
			},
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": "fake-role",
				rbacapi.AnnotationKeyManaged:         rbacapi.AnnotationValueTrue,
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
		Data: map[string][]byte{"token": []byte("fake-token-value")},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/api-tokens/"+testToken.Name,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Secret does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "gets token",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testToken,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Secret in the response
					secret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), secret)
					require.NoError(t, err)
					require.Equal(t, testToken.Name, secret.Name)
				},
			},
		},
	)
}

func Test_server_getSystemAPIToken(t *testing.T) {
	testToken := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testKargoNamespace,
			Name:      "fake-system-token",
			Labels: map[string]string{
				rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
			},
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": "fake-role",
				rbacapi.AnnotationKeyManaged:         rbacapi.AnnotationValueTrue,
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
		Data: map[string][]byte{"token": []byte("fake-token-value")},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/system/api-tokens/"+testToken.Name,
		[]restTestCase{
			{
				name: "Secret does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "gets token",
				clientBuilder: fake.NewClientBuilder().WithObjects(testToken),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Secret in the response
					secret := &corev1.Secret{}
					err := json.Unmarshal(w.Body.Bytes(), secret)
					require.NoError(t, err)
					require.Equal(t, testToken.Name, secret.Name)
				},
			},
		},
	)
}
