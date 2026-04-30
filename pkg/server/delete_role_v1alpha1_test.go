package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_deleteProjectRole(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testKargoRole := &rbacapi.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-role",
			Namespace: testProject.Name,
		},
	}
	testSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      testKargoRole.Name,
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
		http.MethodDelete, "/v1beta1/projects/"+testProject.Name+"/roles/"+testKargoRole.Name,
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
				name: "ServiceAccount exists but is not Kargo-managed",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					func() *corev1.ServiceAccount {
						sa := testSA.DeepCopy()
						delete(sa.Annotations, rbacapi.AnnotationKeyManaged)
						return sa
					}(),
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "deletes Kargo Role",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testSA,
					testRB,
					testRole,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					// Verify the underlying ServiceAccount was deleted from the cluster
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testKargoRole),
						&corev1.ServiceAccount{},
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))

					// Verify the underlying RoleBinding was deleted from the cluster
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testKargoRole),
						&rbacv1.RoleBinding{},
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))

					// Verify the underlying Role was deleted from the cluster
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testKargoRole),
						&rbacv1.Role{},
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))
				},
			},
		},
	)
}
