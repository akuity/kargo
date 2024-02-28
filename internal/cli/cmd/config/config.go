package config

import (
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/config"
)

func NewCommand(cfg config.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage Kargo CLI configuration",
	}

	// Subcommands
	cmd.AddCommand(newSetProjectCommand(cfg))
	cmd.AddCommand(newSetCommand(cfg))
	cmd.AddCommand(newUnsetCommand(cfg))
	return cmd
}
