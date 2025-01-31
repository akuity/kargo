package update

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
)

func NewCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update SUBCOMMAND",
		Short: "Update a resource",
		Args:  option.NoArgs,
	}

	// Register subcommands.
	cmd.AddCommand(newUpdateCredentialsCommand(cfg, streams))
	cmd.AddCommand(newUpdateFreightAliasCommand(cfg))

	return cmd
}
