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

func Test_server_listProjectConfigMaps(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/configmaps",
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "no ConfigMaps exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &corev1.ConfigMapList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists ConfigMaps",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "cm-1",
						},
					},
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProject.Name,
							Name:      "cm-2",
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ConfigMaps in the response
					configs := &corev1.ConfigMapList{}
					err := json.Unmarshal(w.Body.Bytes(), configs)
					require.NoError(t, err)
					require.Len(t, configs.Items, 2)
				},
			},
		},
	)
}

func Test_server_listSystemConfigMaps(t *testing.T) {
	testRESTEndpoint(
		t, &config.ServerConfig{SystemResourcesNamespace: testSystemResourcesNamespace},
		http.MethodGet, "/v1beta1/system/configmaps",
		[]restTestCase{
			{
				name: "no ConfigMaps exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &corev1.ConfigMapList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists ConfigMaps",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testSystemResourcesNamespace,
							Name:      "cm-1",
						},
					},
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testSystemResourcesNamespace,
							Name:      "cm-2",
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ConfigMaps in the response
					configs := &corev1.ConfigMapList{}
					err := json.Unmarshal(w.Body.Bytes(), configs)
					require.NoError(t, err)
					require.Len(t, configs.Items, 2)
				},
			},
		},
	)
}

func Test_server_listSharedConfigMaps(t *testing.T) {
	testRESTEndpoint(
		t, &config.ServerConfig{SharedResourcesNamespace: testSharedResourcesNamespace},
		http.MethodGet, "/v1beta1/shared/configmaps",
		[]restTestCase{
			{
				name: "no ConfigMaps exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &corev1.ConfigMapList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists ConfigMaps",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testSharedResourcesNamespace,
							Name:      "cm-1",
						},
					},
					&corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testSharedResourcesNamespace,
							Name:      "cm-2",
						},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the ConfigMaps in the response
					configs := &corev1.ConfigMapList{}
					err := json.Unmarshal(w.Body.Bytes(), configs)
					require.NoError(t, err)
					require.Len(t, configs.Items, 2)
				},
			},
		},
	)
}
