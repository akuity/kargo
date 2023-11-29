package config

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
)

func newSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set an individual value",
		Args:  option.MinimumNArgs(2),
		Example: `
# Set a default project
kargo config set project my-project
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadCLIConfig()
			if err != nil {
				return errors.Wrap(err, "load cli config")
			}

			key := strings.ToLower(args[0])
			switch key {
			case "project":
				cfg.Project = args[1]
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
