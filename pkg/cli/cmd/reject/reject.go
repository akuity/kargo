package reject

import (
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/option"
)

func NewCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reject SUBCOMMAND",
		Short: "Reject resources",
		Args:  option.NoArgs,
	}

	cmd.AddCommand(newFreightCommand(cfg, streams))

	return cmd
}
