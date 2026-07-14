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

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_deleteProjectConfigMap(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-configmap",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodDelete, "/v1beta1/projects/"+testProject.Name+"/configmaps/"+testConfigMap.Name,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "ConfigMap does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "deletes ConfigMap",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testConfigMap,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					// Verify the ConfigMap was deleted from the cluster
					cm := &corev1.ConfigMap{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testConfigMap),
						cm,
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))
				},
			},
		},
	)
}

func Test_server_deleteSystemConfigMap(t *testing.T) {
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testSystemResourcesNamespace,
			Name:      "fake-configmap",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{SystemResourcesNamespace: testSystemResourcesNamespace},
		http.MethodDelete, "/v1beta1/system/configmaps/"+testConfigMap.Name,
		[]restTestCase{
			{
				name: "ConfigMap does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "deletes ConfigMap",
				clientBuilder: fake.NewClientBuilder().WithObjects(testConfigMap),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					// Verify the ConfigMap was deleted from the cluster
					cm := &corev1.ConfigMap{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testConfigMap),
						cm,
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))
				},
			},
		},
	)
}

func Test_server_deleteSharedConfigMap(t *testing.T) {
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testSharedResourcesNamespace,
			Name:      "fake-configmap",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{SharedResourcesNamespace: testSharedResourcesNamespace},
		http.MethodDelete, "/v1beta1/shared/configmaps/"+testConfigMap.Name,
		[]restTestCase{
			{
				name: "ConfigMap does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "deletes ConfigMap",
				clientBuilder: fake.NewClientBuilder().WithObjects(testConfigMap),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					// Verify the ConfigMap was deleted from the cluster
					cm := &corev1.ConfigMap{}
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testConfigMap),
						cm,
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))
				},
			},
		},
	)
}
