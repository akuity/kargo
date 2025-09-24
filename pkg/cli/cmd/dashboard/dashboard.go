package dashboard

import (
	"errors"
	"fmt"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
)

type dashboardOptions struct {
	Config config.CLIConfig
}

func NewCommand(cfg config.CLIConfig) *cobra.Command {
	cmdOpts := &dashboardOptions{Config: cfg}

	return &cobra.Command{
		Use:   "dashboard",
		Short: "Open the Kargo Dashboard in your default browser",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Open the Kargo Dashboard in the browser
kargo dashboard
`),
		RunE: func(*cobra.Command, []string) error {
			if err := cmdOpts.validate(); err != nil {
				return err
			}
			return cmdOpts.run()
		},
	}
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *dashboardOptions) validate() error {
	if o.Config.APIAddress == "" {
		return errors.New(
			"seems like you are not logged in; please use `kargo login` to authenticate",
		)
	}
	return nil
}

// run opens the Kargo Dashboard in the default browser.
func (o *dashboardOptions) run() error {
	if err := browser.OpenURL(o.Config.APIAddress); err != nil {
		return fmt.Errorf("error opening dashboard in default browser: %w", err)
	}
	return nil
}
