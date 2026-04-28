package mcp

import (
	"github.com/spf13/cobra"

	clicfg "github.com/akuity/kargo/pkg/cli/config"
)

func NewCommand(cfg clicfg.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP server commands",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.HelpFunc()(cmd, nil)
		},
	}
	cmd.AddCommand(newServeCommand(cfg))
	return cmd
}
