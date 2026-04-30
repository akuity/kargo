package logout

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
)

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Log out of the Kargo API server",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Log out of the current Kargo API server
kargo logout
`),
		RunE: func(*cobra.Command, []string) error {
			if err := config.DeleteCLIConfig(); err != nil {
				return fmt.Errorf("error deleting CLI configuration: %w", err)
			}
			return nil
		},
	}
}
