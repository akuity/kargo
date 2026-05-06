package kargomcp

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/cli/config"
)

func TestToolRegistration(t *testing.T) {
	t.Parallel()
	s := New(config.CLIConfig{})

	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go s.mcpServer.Run(ctx, serverTransport) //nolint:errcheck

	session, err := mcp.NewClient(
		&mcp.Implementation{Name: "test"},
		nil,
	).Connect(ctx, clientTransport, nil)
	require.NoError(t, err)

	result, err := session.ListTools(ctx, nil)
	require.NoError(t, err)

	names := make([]string, len(result.Tools))
	for i, tool := range result.Tools {
		names[i] = tool.Name
	}
	require.ElementsMatch(t, []string{
		"get_version_info",
		"list_projects", "get_project",
		"list_stages", "get_stage", "get_stage_freight_history",
		"refresh_stage", "promote_to_stage", "promote_downstream",
		"list_warehouses", "get_warehouse", "refresh_warehouse",
		"list_freight", "get_freight", "approve_freight",
		"list_promotions", "get_promotion", "abort_promotion",
		"list_promotion_tasks", "list_cluster_promotion_tasks",
	}, names)
}

func TestResolveProject(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		cfgProj  string
		explicit string
		assert   func(*testing.T, string, error)
	}{
		{
			name:     "explicit argument is used as-is",
			cfgProj:  "default-proj",
			explicit: "explicit-proj",
			assert: func(t *testing.T, got string, err error) {
				require.NoError(t, err)
				require.Equal(t, "explicit-proj", got)
			},
		},
		{
			name:    "falls back to configured default",
			cfgProj: "default-proj",
			assert: func(t *testing.T, got string, err error) {
				require.NoError(t, err)
				require.Equal(t, "default-proj", got)
			},
		},
		{
			name: "errors when neither explicit nor default is set",
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "project is required")
				require.ErrorContains(t, err, "kargo config set-project")
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := &Server{cfg: config.CLIConfig{Project: tc.cfgProj}}
			got, err := s.resolveProject(tc.explicit)
			tc.assert(t, got, err)
		})
	}
}
