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

func Test_server_listProjectRoles(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/roles",
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "no Kargo Roles exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					roles := []rbacapi.Role{}
					err := json.Unmarshal(w.Body.Bytes(), &roles)
					require.NoError(t, err)
					require.Empty(t, roles)
				},
			},
			{
				name: "lists Kargo Roles",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					&corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "role-1",
						},
					},
					&corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "role-2",
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Kargo Roles in the response
					kargoRoles := []rbacapi.Role{}
					err := json.Unmarshal(w.Body.Bytes(), &kargoRoles)
					require.NoError(t, err)
					require.Len(t, kargoRoles, 2)
					require.Equal(t, "role-1", kargoRoles[0].Name)
					require.Equal(t, "role-2", kargoRoles[1].Name)
				},
			},
		},
	)
}

func Test_server_listSystemRoles(t *testing.T) {
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/system/roles",
		[]restTestCase{
			{
				name:          "no Kargo Roles exist",
				clientBuilder: fake.NewClientBuilder(),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					roles := []rbacapi.Role{}
					err := json.Unmarshal(w.Body.Bytes(), &roles)
					require.NoError(t, err)
					require.Empty(t, roles)
				},
			},
			{
				name: "lists Kargo Roles",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testKargoNamespace,
							Name:      "role-1",
							Labels: map[string]string{
								rbacapi.LabelKeySystemRole: rbacapi.LabelValueTrue,
							},
						},
					},
					&corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testKargoNamespace,
							Name:      "role-2",
							Labels: map[string]string{
								rbacapi.LabelKeySystemRole: rbacapi.LabelValueTrue,
							},
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Kargo Roles in the response
					kargoRoles := []rbacapi.Role{}
					err := json.Unmarshal(w.Body.Bytes(), &kargoRoles)
					require.NoError(t, err)
					require.Len(t, kargoRoles, 2)
					require.Equal(t, "role-1", kargoRoles[0].Name)
					require.Equal(t, "role-2", kargoRoles[1].Name)
				},
			},
		},
	)
}
