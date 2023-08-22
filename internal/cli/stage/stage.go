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
	cmd.AddCommand(newPromoteCommand(opt))
	cmd.AddCommand(newEnableAutoPromotion(opt))
	cmd.AddCommand(newDisableAutoPromotion(opt))
	return cmd
}
