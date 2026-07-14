package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
)

func Test_prepareOutboundWarehouse(t *testing.T) {
	testCases := []struct {
		name       string
		warehouse  *kargoapi.Warehouse
		assertions func(*testing.T, *kargoapi.Warehouse, error)
	}{
		{
			name: "clears internal and creates external",
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{
					InternalSubscriptions: []kargoapi.RepoSubscription{
						{Git: &kargoapi.GitSubscription{}},
						{Image: &kargoapi.ImageSubscription{}},
						{Chart: &kargoapi.ChartSubscription{}},
						{Subscription: &kargoapi.Subscription{SubscriptionType: "generic"}},
					},
				},
			},
			assertions: func(t *testing.T, w *kargoapi.Warehouse, err error) {
				require.NoError(t, err)
				require.Equal(t, 0, len(w.Spec.InternalSubscriptions))
				require.Greater(t, len(w.Spec.Subscriptions), 0)
			},
		},
		{
			name: "empty spec no changes",
			warehouse: &kargoapi.Warehouse{
				Spec: kargoapi.WarehouseSpec{},
			},
			assertions: func(t *testing.T, w *kargoapi.Warehouse, err error) {
				require.NoError(t, err)
				require.Equal(t, 0, len(w.Spec.InternalSubscriptions))
				require.Equal(t, 0, len(w.Spec.Subscriptions))
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := prepareOutboundWarehouse(tc.warehouse)
			tc.assertions(t, tc.warehouse, err)
		})
	}
}

func Test_server_getWarehouse(t *testing.T) {
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
		http.MethodGet, "/v1beta1/projects/"+testProject.Name+"/warehouses/"+testWarehouse.Name,
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
				name: "gets Warehouse",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					testProject,
					testWarehouse,
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Warehouse in the response
					warehouse := &kargoapi.Warehouse{}
					err := json.Unmarshal(w.Body.Bytes(), warehouse)
					require.NoError(t, err)
					require.Equal(t, testProject.Name, warehouse.Namespace)
					require.Equal(t, testWarehouse.Name, warehouse.Name)
				},
			},
		},
	)
}

func Test_server_getWarehouse_watch(t *testing.T) {
	const projectName = "fake-project"
	const warehouseName = "fake-warehouse"

	testRESTWatchEndpoint(
		t, &config.ServerConfig{},
		"/v1beta1/projects/"+projectName+"/warehouses/"+warehouseName+"?watch=true",
		[]restWatchTestCase{
			{
				name: "warehouse not found",
				url:  "/v1beta1/projects/" + projectName + "/warehouses/non-existent?watch=true",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusNotFound, w.Code)
				},
			},
			{
				name: "watches warehouse successfully",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{
						ObjectMeta: metav1.ObjectMeta{Name: projectName},
					},
					&kargoapi.Warehouse{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: projectName,
							Name:      warehouseName,
						},
					},
				),
				operations: func(ctx context.Context, c client.Client) {
					// Fetch the current warehouse first to get the resource version
					warehouse := &kargoapi.Warehouse{}
					_ = c.Get(ctx, client.ObjectKey{Namespace: projectName, Name: warehouseName}, warehouse)

					// Update the warehouse to trigger a watch event
					warehouse.Spec.FreightCreationPolicy = kargoapi.FreightCreationPolicyAutomatic
					_ = c.Update(ctx, warehouse)
				},
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Verify SSE headers
					require.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
					require.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
					require.Equal(t, "keep-alive", w.Header().Get("Connection"))

					// The response body should contain SSE events from the update operation
					body := w.Body.String()
					require.Contains(t, body, "data:")
				},
			},
		},
	)
}
