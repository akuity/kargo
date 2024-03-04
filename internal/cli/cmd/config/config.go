package config

import (
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
)

func NewCommand(cfg config.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config SUBCOMMAND",
		Short: "Manage Kargo CLI configuration",
		Args:  option.NoArgs,
	}

	// Register subcommands.
	cmd.AddCommand(newSetProjectCommand(cfg))

	return cmd
}
