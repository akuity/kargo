package config

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
)

type setProjectOptions struct {
	Config config.CLIConfig

	Project string
}

func newSetProjectCommand(cfg config.CLIConfig) *cobra.Command {
	cmdOpts := &setProjectOptions{Config: cfg}

	cmd := &cobra.Command{
		Use:   "set-project NAME",
		Short: "Set the default project",
		Args:  option.ExactArgs(1),
		Example: `
# Set a default project
kargo config set-project my-project

# Unset a default project
kargo config set-project ""
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdOpts.complete(args)

			return cmdOpts.run()
		},
	}
	return cmd
}

// complete sets the options from the command arguments.
func (o *setProjectOptions) complete(args []string) {
	o.Project = strings.TrimSpace(strings.ToLower(args[0]))
}

// run sets the default project in the CLI configuration using the provided
// options.
func (o *setProjectOptions) run() error {
	o.Config.Project = o.Project

	if err := config.SaveCLIConfig(o.Config); err != nil {
		return fmt.Errorf("save cli config: %w", err)
	}
	return nil
}
