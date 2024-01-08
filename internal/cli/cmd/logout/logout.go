package logout

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/config"
)

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "logout",
		Short:   "Log out of the Kargo API server",
		Example: "kargo logout",
		RunE: func(*cobra.Command, []string) error {
			return errors.Wrap(
				config.DeleteCLIConfig(),
				"error deleting CLI configuration",
			)
		},
	}
}
