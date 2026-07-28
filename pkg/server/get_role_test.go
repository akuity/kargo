package server

import (
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

func Test_server_getProjectRole(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testKargoRole := rbacapi.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-role",
			Namespace: testProject.Name,
		},
		Claims: []rbacapi.Claim{
			{Name: "email", Values: []string{"test@example.com"}},
			{Name: "groups", Values: []string{"admins", "developers"}},
		},
	}
	testSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      testKargoRole.Name,
			Annotations: map[string]string{
				rbacapi.AnnotationKeyManaged:    rbacapi.AnnotationValueTrue,
				rbacapi.AnnotationKeyOIDCClaims: `{"email": ["test@example.com"], "groups": ["admins", "developers"]}`,
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
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/roles/"+testKargoRole.Name,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Kargo Role does not exist", // Underlying ServiceAccount does not exist
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "gets Kargo Role",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testSA,
					testRB,
					testRole,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Kargo Role in the response
					kargoRole := &rbacapi.Role{}
					err := json.Unmarshal(w.Body.Bytes(), kargoRole)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, kargoRole.Namespace)
					require.Equal(t, testRole.Name, kargoRole.Name)
					require.Equal(t, testKargoRole.Claims, kargoRole.Claims)
				},
			},
		},
	)
}

func Test_server_getSystemRole(t *testing.T) {
	testKargoRole := rbacapi.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-role",
			Namespace: testKargoNamespace,
		},
		Claims: []rbacapi.Claim{
			{Name: "email", Values: []string{"test@example.com"}},
			{Name: "groups", Values: []string{"admins", "developers"}},
		},
	}
	testSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testKargoNamespace,
			Name:      testKargoRole.Name,
			Labels: map[string]string{
				rbacapi.LabelKeySystemRole: rbacapi.LabelValueTrue,
			},
			Annotations: map[string]string{
				rbacapi.AnnotationKeyManaged:    rbacapi.AnnotationValueTrue,
				rbacapi.AnnotationKeyOIDCClaims: `{"email": ["test@example.com"], "groups": ["admins", "developers"]}`,
			},
		},
	}
	testRB := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testKargoNamespace,
			Name:      testKargoRole.Name,
			Annotations: map[string]string{
				rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
			},
		},
		Subjects: []rbacv1.Subject{{
			Kind:      rbacv1.ServiceAccountKind,
			Namespace: testKargoNamespace,
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
			Namespace: testKargoNamespace,
			Name:      testKargoRole.Name,
			Annotations: map[string]string{
				rbacapi.AnnotationKeyManaged: rbacapi.AnnotationValueTrue,
			},
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/system/roles/"+testKargoRole.Name,
		[]restTestCase{
			{
				name:          "Kargo Role does not exist", // Underlying ServiceAccount does not exist
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "gets Kargo Role",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testSA,
					testRB,
					testRole,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Kargo Role in the response
					kargoRole := &rbacapi.Role{}
					err := json.Unmarshal(w.Body.Bytes(), kargoRole)
					require.NoError(t, err)
					require.Equal(t, testKargoNamespace, kargoRole.Namespace)
					require.Equal(t, testRole.Name, kargoRole.Name)
					require.Equal(t, testKargoRole.Claims, kargoRole.Claims)
				},
			},
		},
	)
}
