package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_updateRole(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testKargoRole := rbacapi.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-role",
			Namespace: testProject.Name,
			Annotations: map[string]string{
				rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
			},
		},
		Claims: []rbacapi.Claim{
			{Name: "email", Values: []string{"admin@example.com"}},
			{Name: "groups", Values: []string{"admins"}},
		},
	}
	testSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testKargoRole.Name,
			Namespace: testProject.Name,
			Annotations: map[string]string{
				rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
			},
		},
	}
	testRB := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      testKargoRole.Name,
			Annotations: map[string]string{
				rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
			},
		},
		Subjects: []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Namespace: testProject.Name,
			Name:      testKargoRole.Name,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     testKargoRole.Name,
		},
	}
	testRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      testKargoRole.Name,
			Annotations: map[string]string{
				rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
			},
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodPut, "/v1beta1/projects/"+testProject.Name+"/roles/"+testKargoRole.Name, []restTestCase{
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
				name: "role name in URL does not match role name in request body",
				body: mustJSONBody(rbacapi.Role{
					ObjectMeta: metav1.ObjectMeta{Name: "different-name"},
				}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name:          "role does not exist",
				body:          mustJSONBody(testKargoRole),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "updates Kargo Role",
				body: mustJSONBody(func() rbacapi.Role {
					updatedRole := testKargoRole.DeepCopy()
					updatedRole.Claims = []rbacapi.Claim{
						{Name: "email", Values: []string{"admin@example.com"}},
						{Name: "groups", Values: []string{"admins", "devops"}},
					}
					updatedRole.Rules = []rbacv1.PolicyRule{{
						APIGroups: []string{kargoapi.GroupVersion.Group},
						Resources: []string{"stages"},
						Verbs:     []string{"get", "list", "watch"},
					}}
					return *updatedRole
				}()),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testSA,
					testRB,
					testRole,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the response
					var res rbacapi.Role
					err := json.Unmarshal(w.Body.Bytes(), &res)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, res.Namespace)
					require.Equal(t, testKargoRole.Name, res.Name)
					require.True(t, res.KargoManaged)
					require.Equal(
						t,
						[]rbacapi.Claim{
							{Name: "email", Values: []string{"admin@example.com"}},
							{Name: "groups", Values: []string{"admins", "devops"}},
						},
						res.Claims,
					)
					require.Equal(
						t,
						[]rbacv1.PolicyRule{{
							APIGroups: []string{kargoapi.GroupVersion.Group},
							Resources: []string{"stages"},
							Verbs:     []string{"get", "list", "watch"},
						}},
						res.Rules,
					)

					// Verify the underlying Role was updated
					role := &rbacv1.Role{}
					err = c.Get(t.Context(), client.ObjectKeyFromObject(&res), role)
					require.NoError(t, err)
					require.NotEmpty(t, role.Rules)
				},
			},
		},
	)
}
