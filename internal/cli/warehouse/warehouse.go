package warehouse

import (
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/option"
)

func NewCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "warehouse",
		Short: "Manage warehouse",
	}
	cmd.AddCommand(newRefreshCommand(opt))
	return cmd
}
