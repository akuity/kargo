package dashboard

import (
	"github.com/bacongobbler/browser"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/config"
)

func NewCommand(cfg config.CLIConfig) *cobra.Command {
	return &cobra.Command{
		Use:     "dashboard",
		Short:   "Open the Kargo Dashboard in your default browser.",
		Example: "kargo logout",
		RunE: func(*cobra.Command, []string) error {
			if cfg.APIAddress == "" {
				return errors.New(
					"seems like you are not logged in; please use `kargo login` to authenticate",
				)
			}

			return errors.Wrap(
				browser.Open(cfg.APIAddress),
				"error opening dashboard in default browser",
			)
		},
	}
}
