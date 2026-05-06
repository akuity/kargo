package kargomcp

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/akuity/kargo/internal/kargomcp"
	"github.com/akuity/kargo/pkg/cli/config"
)

// RegisterTools registers all OSS Kargo MCP tools onto mcpServer using the
// provided CLI config for authentication and default-project resolution.
//
// EE callers use this to subsume the OSS tool set into their own server so
// that users configure a single MCP server and get both OSS and EE tools.
func RegisterTools(mcpServer *mcp.Server, cfg config.CLIConfig) {
	kargomcp.NewWithExistingServer(mcpServer, cfg)
}
