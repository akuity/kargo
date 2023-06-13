package env

import (
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/option"
)

func NewCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "environment",
		Aliases: []string{"env"},
		Short:   "Manage environments",
	}
	cmd.AddCommand(newPromoteCommand(opt))
	return cmd
}
