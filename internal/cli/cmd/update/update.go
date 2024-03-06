package update

import (
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
)

func NewCommand(cfg config.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update SUBCOMMAND",
		Short: "Update a resource",
		Args:  option.NoArgs,
		Example: `
# Update the alias of a freight for a specified project
kargo update freight --project=my-project abc123 --alias=my-new-alias
`,
	}

	// Register subcommands.
	cmd.AddCommand(newUpdateFreightAliasCommand(cfg))

	return cmd
}
