package update

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/option"
)

func NewCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update SUBCOMMAND",
		Short: "Update a resource",
		Args:  option.NoArgs,
	}

	// Register subcommands.
	cmd.AddCommand(newUpdateConfigMapCommand(cfg, streams))
	cmd.AddCommand(newUpdateGenericCredentialsCommand(cfg, streams))
	cmd.AddCommand(newUpdateRepoCredentialsCommand(cfg, streams))
	cmd.AddCommand(newUpdateFreightAliasCommand(cfg))

	return cmd
}
