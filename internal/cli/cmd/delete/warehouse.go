package delete

import (
	"context"
	goerrors "errors"
	"slices"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type deleteWarehouseOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
	Names   []string
}

func newWarehouseCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &deleteWarehouseOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("deleted").WithTypeSetter(runtime.NewScheme()),
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
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.PrintFlags.AddFlags(cmd)

	option.Project(cmd.Flags(), &o.Project, o.Config.Project,
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
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return errors.Wrap(err, "get client from config")
	}

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return errors.Wrap(err, "create printer")
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
		_ = printer.PrintObj(&kargoapi.Warehouse{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: o.Project,
			},
		}, o.IOStreams.Out)
	}
	return resErr
}
