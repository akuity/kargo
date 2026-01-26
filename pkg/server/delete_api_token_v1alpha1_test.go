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

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_deleteProjectAPIToken(t *testing.T) {
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
				"kubernetes.io/service-account.name": "fake-service-account",
				rbacapi.AnnotationKeyManaged:         rbacapi.AnnotationValueTrue,
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodDelete, "/v1beta1/projects/"+testProject.Name+"/api-tokens/"+testToken.Name,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "token does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not a ServiceAccount token",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					func() *corev1.Secret {
						secret := testToken.DeepCopy()
						secret.Type = corev1.SecretTypeOpaque
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "Secret exists but is not annotated with ServiceAccount",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					func() *corev1.Secret {
						secret := testToken.DeepCopy()
						delete(secret.Annotations, "kubernetes.io/service-account.name")
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "Secret exists but is not a Kargo API token",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					func() *corev1.Secret {
						secret := testToken.DeepCopy()
						delete(secret.Labels, rbacapi.LabelKeyAPIToken)
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "token exists but is not Kargo-managed",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					func() *corev1.Secret {
						secret := testToken.DeepCopy()
						delete(secret.Annotations, rbacapi.AnnotationKeyManaged)
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "deletes token",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testToken,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					// Verify the Secret was deleted from the cluster
					secret := &corev1.Secret{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testToken),
						secret,
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))
				},
			},
		},
	)
}

func Test_server_deleteSystemAPIToken(t *testing.T) {
	testToken := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testKargoNamespace,
			Name:      "fake-token",
			Labels: map[string]string{
				rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
			},
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": "fake-service-account",
				rbacapi.AnnotationKeyManaged:         rbacapi.AnnotationValueTrue,
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodDelete, "/v1beta1/system/api-tokens/"+testToken.Name,
		[]restTestCase{
			{
				name: "token does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "Secret exists but is not a ServiceAccount token",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					func() *corev1.Secret {
						secret := testToken.DeepCopy()
						secret.Type = corev1.SecretTypeOpaque
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "Secret exists but is not annotated with ServiceAccount",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					func() *corev1.Secret {
						secret := testToken.DeepCopy()
						delete(secret.Annotations, "kubernetes.io/service-account.name")
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "Secret exists but is not a Kargo API token",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					func() *corev1.Secret {
						secret := testToken.DeepCopy()
						delete(secret.Labels, rbacapi.LabelKeyAPIToken)
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "token exists but is not Kargo-managed",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					func() *corev1.Secret {
						secret := testToken.DeepCopy()
						delete(secret.Annotations, rbacapi.AnnotationKeyManaged)
						return secret
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name: "deletes token",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testToken,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					// Verify the Secret was deleted from the cluster
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testToken),
						&corev1.Secret{},
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))
				},
			},
		},
	)
}
