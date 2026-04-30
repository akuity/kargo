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
)

func Test_server_createProjectRole(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testKargoRole := rbacapi.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-role",
			Namespace: testProject.Name,
		},
		Claims: []rbacapi.Claim{
			{Name: "email", Values: []string{"admin@example.com"}},
			{Name: "groups", Values: []string{"admins", "developers"}},
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{""},
			Resources: []string{"stages"},
			Verbs:     []string{"*"},
		}},
	}
	testSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testKargoRole.Name,
			Namespace: testKargoRole.Namespace,
		},
	}
	testRESTEndpoint(
		t, nil,
		http.MethodPost, "/v1beta1/projects/"+testProject.Name+"/roles",
		[]restTestCase{
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
				name:          "missing name in request body",
				body:          mustJSONBody(rbacapi.Role{ObjectMeta: metav1.ObjectMeta{}}),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusBadRequest, w.Code)
				},
			},
			{
				name: "Kargo Role already exists", // Because the underlying ServiceAccount exists
				body: mustJSONBody(testKargoRole),
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testSA,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusConflict, w.Code)
				},
			},
			{
				name:          "creates Kargo Role",
				body:          mustJSONBody(testKargoRole),
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusCreated, w.Code)

					// Examine the Kargo Role in the response
					resKargoRole := &rbacapi.Role{}
					err := json.Unmarshal(w.Body.Bytes(), &resKargoRole)
					require.NoError(t, err)
					require.Equal(t, testKargoRole.Name, resKargoRole.Name)

					// Verify the underlying resources were created in the cluster
					sa := &corev1.ServiceAccount{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resKargoRole),
						sa,
					)
					require.NoError(t, err)

					rb := &rbacv1.RoleBinding{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resKargoRole),
						rb,
					)
					require.NoError(t, err)

					role := &rbacv1.Role{}
					err = c.Get(
						t.Context(),
						client.ObjectKeyFromObject(resKargoRole),
						role,
					)
					require.NoError(t, err)
				},
			},
		},
	)
}
