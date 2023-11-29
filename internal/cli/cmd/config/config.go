package config

import "github.com/spf13/cobra"

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage Kargo CLI configuration",
	}

	// Subcommands
	cmd.AddCommand(newSetCommand())
	cmd.AddCommand(newUnsetCommand())
	return cmd
}
