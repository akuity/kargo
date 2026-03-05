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

func Test_server_listProjectAPITokens(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}

	var testURL = "/v1beta1/projects/" + testProject.Name + "/api-tokens"
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, testURL,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "no tokens exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					tokens := corev1.SecretList{}
					err := json.Unmarshal(w.Body.Bytes(), &tokens)
					require.NoError(t, err)
					require.Empty(t, tokens)
				},
			},
			{
				name: "lists tokens",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "token-1",
							Labels: map[string]string{
								rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
							},
							Annotations: map[string]string{
								"kubernetes.io/service-account.name": "role-1",
							},
						},
						Type: corev1.SecretTypeServiceAccountToken,
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "token-2",
							Labels: map[string]string{
								rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
							},
							Annotations: map[string]string{
								"kubernetes.io/service-account.name": "role-2",
							},
						},
						Type: corev1.SecretTypeServiceAccountToken,
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
			{
				name: "lists tokens filtered by Role",
				url:  testURL + "?role=role-1",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					&corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "role-1",
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "token-1",
							Labels: map[string]string{
								rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
							},
							Annotations: map[string]string{
								"kubernetes.io/service-account.name": "role-1",
							},
						},
						Type: corev1.SecretTypeServiceAccountToken,
					},
					&corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "role-2",
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "token-2",
							Labels: map[string]string{
								rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
							},
							Annotations: map[string]string{
								"kubernetes.io/service-account.name": "role-2",
							},
						},
						Type: corev1.SecretTypeServiceAccountToken,
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Secrets in the response
					secrets := corev1.SecretList{}
					err := json.Unmarshal(w.Body.Bytes(), &secrets)
					require.NoError(t, err)
					require.Len(t, secrets.Items, 1)
					require.Equal(t, "token-1", secrets.Items[0].Name)
				},
			},
		},
	)
}

func Test_server_listSystemAPITokens(t *testing.T) {
	const testURL = "/v1beta1/system/api-tokens"
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, testURL,
		[]restTestCase{
			{
				name:          "no tokens exist",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					tokens := corev1.SecretList{}
					err := json.Unmarshal(w.Body.Bytes(), &tokens)
					require.NoError(t, err)
					require.Empty(t, tokens.Items)
				},
			},
			{
				name: "lists tokens",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "system-token-1",
							Namespace: testKargoNamespace,
							Labels: map[string]string{
								rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
							},
							Annotations: map[string]string{
								"kubernetes.io/service-account.name": "system-role-1",
							},
						},
						Type: corev1.SecretTypeServiceAccountToken,
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "system-token-2",
							Namespace: testKargoNamespace,
							Labels: map[string]string{
								rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
							},
							Annotations: map[string]string{
								"kubernetes.io/service-account.name": "system-role-2",
							},
						},
						Type: corev1.SecretTypeServiceAccountToken,
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Secrets in the response
					secrets := corev1.SecretList{}
					err := json.Unmarshal(w.Body.Bytes(), &secrets)
					require.NoError(t, err)
					require.Len(t, secrets.Items, 2)
				},
			},
			{
				name: "lists tokens filtered by Role",
				url:  testURL + "?role=system-role-1",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "system-role-1",
							Namespace: testKargoNamespace,
							Labels: map[string]string{
								rbacapi.LabelKeySystemRole: rbacapi.LabelValueTrue,
							},
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "system-token-1",
							Namespace: testKargoNamespace,
							Labels: map[string]string{
								rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
							},
							Annotations: map[string]string{
								"kubernetes.io/service-account.name": "system-role-1",
							},
						},
						Type: corev1.SecretTypeServiceAccountToken,
					},
					&corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "system-sa-2",
							Namespace: testKargoNamespace,
						},
					},
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "system-token-2",
							Namespace: testKargoNamespace,
							Labels: map[string]string{
								rbacapi.LabelKeyAPIToken: rbacapi.LabelValueTrue,
							},
							Annotations: map[string]string{
								"kubernetes.io/service-account.name": "system-role-2",
							},
						},
						Type: corev1.SecretTypeServiceAccountToken,
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Secrets in the response
					secrets := corev1.SecretList{}
					err := json.Unmarshal(w.Body.Bytes(), &secrets)
					require.NoError(t, err)
					require.Len(t, secrets.Items, 1)
					require.Equal(t, "system-token-1", secrets.Items[0].Name)
				},
			},
		},
	)
}
