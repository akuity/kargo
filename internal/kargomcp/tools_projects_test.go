package kargomcp

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandleListProjects(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects": jsonOK(`{"items":[{"metadata":{"name":"my-proj"}}]}`),
	})
	result, _, err := s.handleListProjects(context.Background(), nil, listProjectsArgs{})
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Contains(t, structuredContent(t, result), "my-proj")
}

func TestHandleGetProject(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/projects/test-project": jsonOK(`{"metadata":{"name":"test-project"}}`),
	})
	result, _, err := s.handleGetProject(context.Background(), nil, getProjectArgs{})
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Contains(t, structuredContent(t, result), "test-project")
}
