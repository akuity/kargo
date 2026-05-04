package kargomcp

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandleListWarehouses(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project/warehouses": jsonOK(`{"items":[{"metadata":{"name":"my-wh"}}]}`),
	})
	result, _, err := s.handleListWarehouses(context.Background(), nil, listWarehousesArgs{})
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Contains(t, structuredContent(t, result), "my-wh")
}

func TestHandleGetWarehouse(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project/warehouses/my-wh": jsonOK(`{"metadata":{"name":"my-wh"}}`),
	})
	result, _, err := s.handleGetWarehouse(context.Background(), nil, getWarehouseArgs{Warehouse: "my-wh"})
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Contains(t, structuredContent(t, result), "my-wh")
}

func TestHandleRefreshWarehouse(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project/warehouses/my-wh/refresh": jsonOK(`{}`),
	})
	result, _, err := s.handleRefreshWarehouse(context.Background(), nil, refreshWarehouseArgs{Warehouse: "my-wh"})
	require.NoError(t, err)
	require.False(t, result.IsError)
}
