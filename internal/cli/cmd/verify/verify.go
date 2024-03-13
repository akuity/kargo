package verify

import (
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
)

func NewCommand(cfg config.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify SUBCOMMAND",
		Short: "Verify a stage",
		Args:  option.NoArgs,
		Example: `
# Verify a stage
kargo verify stage --project=my-project my-stage
`,
	}

	// Register subcommands.
	cmd.AddCommand(newVerifyStageCommand(cfg))

	return cmd
}
