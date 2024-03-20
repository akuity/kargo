package config

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
)

func NewCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config SUBCOMMAND",
		Short: "Manage Kargo CLI configuration",
		Args:  option.NoArgs,
	}

	// Register subcommands.
	cmd.AddCommand(newGetProjectCommand(cfg, streams))
	cmd.AddCommand(newSetProjectCommand(cfg))

	return cmd
}
