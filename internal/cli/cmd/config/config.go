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

	// Register subcommands.
	cmd.AddCommand(newSetProjectCommand(cfg))

	return cmd
}
