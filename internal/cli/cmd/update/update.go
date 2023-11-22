package update

import (
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/option"
)

func NewCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a freight alias",
	}
	cmd.AddCommand(newUpdateFreightAliasCommand(opt))
	return cmd
}
