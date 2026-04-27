package kargomcp

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	generatedclient "github.com/akuity/kargo/pkg/client/generated"
)

// Server wraps an MCP server backed by the Kargo REST API client.
type Server struct {
	mcpServer *mcp.Server
	cfg       config.CLIConfig
}

// New creates a new Kargo MCP server with all tools registered.
func New(cfg config.CLIConfig) *Server {
	s := &Server{
		mcpServer: mcp.NewServer(
			&mcp.Implementation{Name: "kargo-mcp", Version: "0.1.0"},
			&mcp.ServerOptions{
				Instructions: "Kargo MCP server for managing continuous promotion pipelines. " +
					"Use these tools to query stages, warehouses, freight, and promotions, " +
					"and to trigger or approve promotions. " +
					"If you receive an auth error, ask the user to run `kargo login`.",
			},
		),
		cfg: cfg,
	}
	s.registerTools()
	return s
}

// Run starts the MCP server on the stdio transport.
func (s *Server) Run(ctx context.Context) error {
	return s.mcpServer.Run(ctx, &mcp.StdioTransport{})
}

func (s *Server) registerTools() {
	s.registerServerTools()
	s.registerProjectTools()
	s.registerStageTools()
	s.registerWarehouseTools()
	s.registerFreightTools()
	s.registerPromotionTools()
}

// apiClient constructs an authenticated Kargo REST API client, handling token
// refresh transparently. Returns a user-friendly error if auth is missing or expired.
func (s *Server) apiClient(ctx context.Context) (*generatedclient.KargoAPI, error) {
	c, err := client.GetClientFromConfig(ctx, s.cfg, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("%w\nRun `kargo login` to authenticate", err)
	}
	return c, nil
}
