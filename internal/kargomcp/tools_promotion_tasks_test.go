package kargomcp

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandleListPromotionTasks(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project/promotion-tasks": jsonOK(`{"items":[{"metadata":{"name":"my-task"}}]}`),
	})
	result, _, err := s.handleListPromotionTasks(context.Background(), nil, listPromotionTasksArgs{})
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Contains(t, structuredContent(t, result), "my-task")
}

func TestHandleListClusterPromotionTasks(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/shared/cluster-promotion-tasks": jsonOK(`{"items":[{"metadata":{"name":"cluster-task"}}]}`),
	})
	result, _, err := s.handleListClusterPromotionTasks(context.Background(), nil, listClusterPromotionTasksArgs{})
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Contains(t, structuredContent(t, result), "cluster-task")
}

func TestHandleGetPromotionTask(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project/promotion-tasks/my-task": jsonOK(`{"metadata":{"name":"my-task"}}`),
	})
	result, _, err := s.handleGetPromotionTask(context.Background(), nil, getPromotionTaskArgs{PromotionTask: "my-task"})
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Contains(t, structuredContent(t, result), "my-task")
}

func TestHandleGetClusterPromotionTask(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/shared/cluster-promotion-tasks/cluster-task": jsonOK(`{"metadata":{"name":"cluster-task"}}`),
	})
	args := getClusterPromotionTaskArgs{PromotionTask: "cluster-task"}
	result, _, err := s.handleGetClusterPromotionTask(context.Background(), nil, args)
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Contains(t, structuredContent(t, result), "cluster-task")
}
