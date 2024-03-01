package dashboard

import (
	"github.com/bacongobbler/browser"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
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
		Example: `
# Open the Kargo Dashboard
kargo dashboard
`,
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
	return errors.Wrap(
		browser.Open(o.Config.APIAddress),
		"error opening dashboard in default browser",
	)
}
