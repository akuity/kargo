package update

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
)

func NewCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update SUBCOMMAND",
		Short: "Update a resource",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Update the alias of a freight for a specified project
kargo update freight --project=my-project --name=abc123 --new-alias=my-new-alias
`),
	}

	// Register subcommands.
	cmd.AddCommand(newUpdateCredentialsCommand(cfg, streams))
	cmd.AddCommand(newUpdateFreightAliasCommand(cfg))

	return cmd
}
