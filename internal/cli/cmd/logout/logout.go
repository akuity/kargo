package logout

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
)

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out of the Kargo API server",
		Args:  option.NoArgs,
		Example: `
# Log out of the current Kargo API server
kargo logout
`,
		RunE: func(*cobra.Command, []string) error {
			return errors.Wrap(
				config.DeleteCLIConfig(),
				"error deleting CLI configuration",
			)
		},
	}
}
