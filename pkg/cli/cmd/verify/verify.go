package verify

import (
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
)

func NewCommand(cfg config.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify SUBCOMMAND",
		Short: "Verify a stage",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Verify a stage
kargo verify stage --project=my-project my-stage
`),
	}

	// Register subcommands.
	cmd.AddCommand(newVerifyStageCommand(cfg))

	return cmd
}
