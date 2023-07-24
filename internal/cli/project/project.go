package project

import (
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/option"
)

func NewCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}
	cmd.AddCommand(newCreateCommand(opt))
	cmd.AddCommand(newDeleteCommand(opt))
	cmd.AddCommand(newListCommand(opt))
	return cmd
}
