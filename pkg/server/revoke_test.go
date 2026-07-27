package server

import (
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

func Test_server_revoke(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	// ServiceAccount is required for the role to exist in rolesDB
	testRoleSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-role",
			Namespace: testProject.Name,
			Annotations: map[string]string{
				rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
			},
		},
	}

	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPost, "/v1beta1/projects/"+testProject.Name+"/roles/revocations",
		[]restTestCase{
			{
				name:          "Project not found",
				clientBuilder: fake.NewClientBuilder(),
				body: mustJSONBody(revokeRequest{
					Role:       "test-role",
					UserClaims: &userClaims{Claims: []rbacapi.Claim{{Name: "sub", Values: []string{"user1"}}}},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Missing role",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				body: mustJSONBody(revokeRequest{
					UserClaims: &userClaims{Claims: []rbacapi.Claim{{Name: "sub", Values: []string{"user1"}}}},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "Missing claims/serviceAccounts/resourceDetails",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				body: mustJSONBody(revokeRequest{
					Role: "test-role",
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "Revoke role from users",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testRoleSA),
				body: mustJSONBody(revokeRequest{
					Role:       "test-role",
					UserClaims: &userClaims{Claims: []rbacapi.Claim{{Name: "sub", Values: []string{"user1"}}}},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
				},
			},
			{
				name:          "Revoke permissions from role",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject, testRoleSA),
				body: mustJSONBody(revokeRequest{
					Role: "test-role",
					ResourceDetails: &rbacapi.ResourceDetails{
						ResourceType: "stages",
						ResourceName: "*",
						Verbs:        []string{"get", "list"},
					},
				}),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
				},
			},
		},
	)
}
