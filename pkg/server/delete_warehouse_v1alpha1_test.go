package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_server_deleteWarehouse(t *testing.T) {
	testProject := &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "fake-project"},
	}
	testWarehouse := &kargoapi.Warehouse{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testProject.Name,
			Name:      "fake-warehouse",
		},
	}
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodDelete, "/v1beta1/projects/"+testProject.Name+"/warehouses/"+testWarehouse.Name,
		[]restTestCase{
			{
				name: "Project does not exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name:          "Warehouse does not exist",
				clientBuilder: fake.NewClientBuilder().WithObjects(testProject),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "deletes Warehouse",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testWarehouse,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, c client.Client) {
					require.Equal(t, http.StatusNoContent, w.Code)

					// Verify the Warehouse was deleted from the cluster
					err := c.Get(
						t.Context(),
						client.ObjectKeyFromObject(testWarehouse),
						&kargoapi.Warehouse{},
					)
					require.Error(t, err)
					require.True(t, apierrors.IsNotFound(err))
				},
			},
		},
	)
}
