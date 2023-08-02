package stage

import (
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/option"
)

func NewCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stage",
		Short: "Manage stages",
	}
	cmd.AddCommand(newDeleteCommand(opt))
	cmd.AddCommand(newListCommand(opt))
	cmd.AddCommand(newPromoteCommand(opt))
	return cmd
}
