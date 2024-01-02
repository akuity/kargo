package config

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
)

func newUnsetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unset",
		Short: "Unset an individual value",
		Args:  option.MinimumNArgs(2),
		Example: `
# Unset a default project
kargo config unset project my-project
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.EnsureCLIConfig()
			if err != nil {
				return errors.Wrap(err, "ensure cli config")
			}

			key := strings.ToLower(args[0])
			switch key {
			case "project":
				cfg.Project = ""
			default:
				return errors.Errorf("unknown key %q", key)
			}

			if err := config.SaveCLIConfig(cfg); err != nil {
				return errors.Wrap(err, "save cli config")
			}
			return nil
		},
	}
	return cmd
}
