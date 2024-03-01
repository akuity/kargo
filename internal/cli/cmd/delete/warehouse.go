package delete

import (
	"context"
	goerrors "errors"
	"fmt"
	"slices"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type deleteWarehouseOptions struct {
	*option.Option
	Config config.CLIConfig

	Names []string
}

func newWarehouseCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmdOpts := &deleteWarehouseOptions{
		Option: opt,
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use:   "warehouse [--project=project] (NAME ...)",
		Short: "Delete warehouse by name",
		Args:  option.MinimumNArgs(1),
		Example: `
# Delete a warehouse
kargo delete warehouse --project=my-project my-warehouse

# Delete multiple warehouses
kargo delete warehouse --project=my-project my-warehouse1 my-warehouse2

# Delete a warehouse in the default project
kargo config set-project my-project
kargo delete warehouse my-warehouse
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdOpts.complete(args)

			if err := cmdOpts.validate(); err != nil {
				return err
			}

			return cmdOpts.run(cmd.Context())
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	return cmd
}

// addFlags adds the flags for the delete warehouse options to the provided
// command.
func (o *deleteWarehouseOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)

	option.Project(cmd.Flags(), &o.Project, o.Project,
		"The Project for which to delete Warehouses. If not set, the default project will be used.")
}

// complete sets the options from the command arguments.
func (o *deleteWarehouseOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *deleteWarehouseOptions) validate() error {
	var errs []error

	if o.Project == "" {
		errs = append(errs, errors.New("project is required"))
	}

	if len(o.Names) == 0 {
		errs = append(errs, errors.New("at least one warehouse name is required"))
	}

	return goerrors.Join(errs...)
}

// run removes the warehouse(s) based on the options.
func (o *deleteWarehouseOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.Option)
	if err != nil {
		return errors.Wrap(err, "get client from config")
	}

	var resErr error
	for _, name := range o.Names {
		if _, err := kargoSvcCli.DeleteWarehouse(
			ctx,
			connect.NewRequest(
				&v1alpha1.DeleteWarehouseRequest{
					Project: o.Project,
					Name:    name,
				},
			),
		); err != nil {
			resErr = goerrors.Join(resErr, errors.Wrap(err, "Error"))
			continue
		}
		_, _ = fmt.Fprintf(o.IOStreams.Out, "Warehouse Deleted: %q\n", name)
	}
	return resErr
}
