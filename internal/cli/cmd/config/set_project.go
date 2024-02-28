package config

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
)

func newSetProjectCommand(cfg config.CLIConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-project",
		Short: "Set the default project",
		Args:  option.ExactArgs(1),
		Example: `
# Set a default project
kargo config set-project my-project

# Unset a default project
kargo config set-project ""
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			project := strings.TrimSpace(strings.ToLower(args[0]))
			cfg.Project = project

			if err := config.SaveCLIConfig(cfg); err != nil {
				return errors.Wrap(err, "save cli config")
			}
			return nil
		},
	}
	return cmd
}
