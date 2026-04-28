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
	instructions := "Kargo MCP server for managing continuous promotion pipelines. " +
		"Use these tools to query stages, warehouses, freight, and promotions, " +
		"and to trigger or approve promotions. " +
		"If you receive an auth error, ask the user to run `kargo login`."
	if cfg.APIAddress != "" {
		instructions += fmt.Sprintf(" Connected to Kargo server at %s.", cfg.APIAddress)
	}
	if cfg.Project != "" {
		instructions += fmt.Sprintf(" Default project: %q.", cfg.Project)
	}
	s := &Server{
		mcpServer: mcp.NewServer(
			&mcp.Implementation{Name: "kargo-mcp", Version: "0.1.0"},
			&mcp.ServerOptions{Instructions: instructions},
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
	s.registerPromotionTaskTools()
}

// resolveProject returns explicit if non-empty, falls back to the configured
// default project, or returns an error if neither is set.
func (s *Server) resolveProject(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if s.cfg.Project != "" {
		return s.cfg.Project, nil
	}
	return "", fmt.Errorf(
		"project is required: pass it as an argument or set a default with `kargo config set-project <project>`",
	)
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
