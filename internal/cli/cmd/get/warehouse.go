package get

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/io"
	"github.com/akuity/kargo/internal/cli/kubernetes"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
)

type getWarehousesOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	*getOptions

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
	Names   []string
}

func newGetWarehousesCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
	getOptions *getOptions,
) *cobra.Command {
	cmdOpts := &getWarehousesOptions{
		Config:     cfg,
		IOStreams:  streams,
		getOptions: getOptions,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "warehouses [--project=project] [NAME ...] [--no-headers]",
		Aliases: []string{"warehouse"},
		Short:   "Display one or many warehouses",
		Example: templates.Example(`
# List all warehouses in my-project
kargo get warehouses --project=my-project

# List all warehouses in my-project in JSON output format
kargo get warehouses --project=my-project -o json

# Get a specific warehouse in my-project
kargo get warehouse --project=my-project my-warehouse

# List all warehouses in the default project
kargo config set-project my-project
kargo get warehouses

# Get a specific warehouse in the default project
kargo config set-project my-project
kargo get warehouse my-warehouse
`),
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

	// Set the input/output streams for the command.
	io.SetIOStreams(cmd, cmdOpts.IOStreams)

	return cmd
}

// addFlags adds the flags for the get warehouses options to the provided
// command.
func (o *getWarehousesOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.PrintFlags.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project for which to list Warehouses. If not set, the default project will be used.",
	)
}

// complete sets the options from the command arguments.
func (o *getWarehousesOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *getWarehousesOptions) validate() error {
	if o.Project == "" {
		return errors.New("project is required")
	}
	return nil
}

// run gets the warehouses from the server and prints them to the console.
func (o *getWarehousesOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	if len(o.Names) == 0 {
		var resp *connect.Response[v1alpha1.ListWarehousesResponse]
		if resp, err = kargoSvcCli.ListWarehouses(
			ctx,
			connect.NewRequest(
				&v1alpha1.ListWarehousesRequest{
					Project: o.Project,
				},
			),
		); err != nil {
			return fmt.Errorf("list warehouses: %w", err)
		}
		return printObjects(resp.Msg.GetWarehouses(), o.PrintFlags, o.IOStreams, o.NoHeaders)
	}

	res := make([]*kargoapi.Warehouse, 0, len(o.Names))
	errs := make([]error, 0, len(o.Names))
	for _, name := range o.Names {
		var resp *connect.Response[v1alpha1.GetWarehouseResponse]
		if resp, err = kargoSvcCli.GetWarehouse(
			ctx,
			connect.NewRequest(
				&v1alpha1.GetWarehouseRequest{
					Project: o.Project,
					Name:    name,
				},
			),
		); err != nil {
			errs = append(errs, err)
			continue
		}
		res = append(res, resp.Msg.GetWarehouse())
	}

	if err = printObjects(res, o.PrintFlags, o.IOStreams, o.NoHeaders); err != nil {
		return fmt.Errorf("print warehouses: %w", err)
	}
	return errors.Join(errs...)
}

func newWarehouseTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		warehouse := item.Object.(*kargoapi.Warehouse) // nolint: forcetypeassert
		rows[i] = metav1.TableRow{
			Cells: []any{
				warehouse.Name,
				warehouse.Spec.Shard,
				duration.HumanDuration(time.Since(warehouse.CreationTimestamp.Time)),
			},
			Object: list.Items[i],
		}
	}
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Shard", Type: "string"},
			{Name: "Age", Type: "string"},
		},
		Rows: rows,
	}
}
