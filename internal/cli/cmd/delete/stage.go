package delete

import (
	goerrors "errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type deleteStageOptions struct {
	*option.Option
}

// addFlags adds the flags for the delete stage options to the provided command.
func (o *deleteStageOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)

	option.Project(cmd.Flags(), &o.Project, o.Project,
		"The Project for which to delete Stages. If not set, the default project will be used.")
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *deleteStageOptions) validate() error {
	if o.Project == "" {
		return errors.New("project is required")
	}
	return nil
}

func newStageCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmdOpts := &deleteStageOptions{Option: opt}

	cmd := &cobra.Command{
		Use:   "stage [NAME]...",
		Short: "Delete stage by name",
		Args:  option.MinimumNArgs(1),
		Example: `
# Delete stage
kargo delete stage --project=my-project my-stage
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if err := cmdOpts.validate(); err != nil {
				return err
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, opt)
			if err != nil {
				return errors.Wrap(err, "get client from config")
			}

			var resErr error
			for _, name := range slices.Compact(args) {
				if _, err := kargoSvcCli.DeleteStage(ctx, connect.NewRequest(&v1alpha1.DeleteStageRequest{
					Project: cmdOpts.Project,
					Name:    name,
				})); err != nil {
					resErr = goerrors.Join(resErr, errors.Wrap(err, "Error"))
					continue
				}
				_, _ = fmt.Fprintf(cmdOpts.IOStreams.Out, "Stage Deleted: %q\n", name)
			}
			return resErr
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	return cmd
}
