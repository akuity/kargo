package option

import (
	"fmt"

	"github.com/spf13/cobra"
)

// ExactArgs is a wrapper around cobra.ExactArgs to additionally print usage string
func ExactArgs(n int) cobra.PositionalArgs {
	exactArgs := cobra.ExactArgs(n)
	return func(cmd *cobra.Command, args []string) error {
		if err := exactArgs(cmd, args); err != nil {
			_, _ = fmt.Fprintf(cmd.OutOrStderr(), "%s\n", cmd.UsageString())
			return err
		}
		return nil
	}
}

// MaximumNArgs is a wrapper around cobra.MaximumNArgs to additionally print usage string
func MaximumNArgs(n int) cobra.PositionalArgs {
	maxNArgs := cobra.MaximumNArgs(n)
	return func(cmd *cobra.Command, args []string) error {
		if err := maxNArgs(cmd, args); err != nil {
			_, _ = fmt.Fprintf(cmd.OutOrStderr(), "%s\n", cmd.UsageString())
			return err
		}
		return nil
	}
}

// MinimumNArgs is a wrapper around cobra.MinimumNArgs to additionally print usage string
func MinimumNArgs(n int) cobra.PositionalArgs {
	minNArgs := cobra.MinimumNArgs(n)
	return func(cmd *cobra.Command, args []string) error {
		if err := minNArgs(cmd, args); err != nil {
			_, _ = fmt.Fprintf(cmd.OutOrStderr(), "%s\n", cmd.UsageString())
			return err
		}
		return nil
	}
}

// NoArgs is a wrapper around cobra.NoArgs to additionally print usage string
func NoArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.NoArgs(cmd, args); err != nil {
		_, _ = fmt.Fprintf(cmd.OutOrStderr(), "%s\n", cmd.UsageString())
		return err
	}
	return nil
}
