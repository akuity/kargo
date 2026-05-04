package kargomcp

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandleGetVersionInfo(t *testing.T) {
	t.Parallel()
	s := newTestServer(t, map[string]http.HandlerFunc{
		"/v1beta1/system/server-version": jsonOK(`{"version":"v1.2.3","gitCommit":"abc123"}`),
	})
	result, _, err := s.handleGetVersionInfo(context.Background(), nil, getVersionInfoArgs{})
	require.NoError(t, err)
	require.False(t, result.IsError)
	require.Contains(t, structuredContent(t, result), "v1.2.3")
}
