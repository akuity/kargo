package option

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/config"
)

type Option struct {
	InsecureTLS        bool
	LocalServerAddress string
	UseLocalServer     bool
}

func NewOption(cfg config.CLIConfig) *Option {
	return &Option{}
}

// ExactArgs is a wrapper around cobra.ExactArgs to additionally print usage string
func ExactArgs(n int) cobra.PositionalArgs {
	exactArgs := cobra.ExactArgs(n)
	return func(cmd *cobra.Command, args []string) error {
		if err := exactArgs(cmd, args); err != nil {
			fmt.Println(cmd.UsageString())
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
			fmt.Println(cmd.UsageString())
			return err
		}
		return nil
	}
}

// NoArgs is a wrapper around cobra.NoArgs to additionally print usage string
func NoArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.NoArgs(cmd, args); err != nil {
		fmt.Println(cmd.UsageString())
		return err
	}
	return nil
}
