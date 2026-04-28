package mcp

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/kargomcp"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	clicfg "github.com/akuity/kargo/pkg/cli/config"
)

func newServeCommand(cfg clicfg.CLIConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the Kargo MCP server (stdio transport)",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Start the MCP server using credentials from kargo login
kargo mcp serve

# Start with explicit server address and token
KARGO_API_ADDRESS=https://kargo.example.com KARGO_API_TOKEN=<token> kargo mcp serve
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintf(os.Stderr, "kargo mcp serve starting (address: %s)\n", cfg.APIAddress)
			return kargomcp.New(cfg).Run(cmd.Context())
		},
	}
}
