package get

import (
	"context"
	goerrors "errors"
	"slices"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type getWarehousesOptions struct {
	*option.Option
	Config config.CLIConfig

	Names []string
}

func newGetWarehousesCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmdOpts := &getWarehousesOptions{
		Option: opt,
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use:     "warehouses [--project=project] [NAME ...]",
		Aliases: []string{"warehouse"},
		Short:   "Display one or many warehouses",
		Example: `
# List all warehouses in the project
kargo get warehouses --project=my-project

# List all warehouses in JSON output format
kargo get warehouses --project=my-project -o json

# Get a warehouse in the project
kargo get warehouses --project=my-project my-warehouse

# List all warehouses in the default project
kargo config set-project my-project
kargo get warehouses
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

// addFlags adds the flags for the get warehouses options to the provided
// command.
func (o *getWarehousesOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)

	option.Project(cmd.Flags(), &o.Project, o.Project,
		"The Project for which to list Warehouses. If not set, the default project will be used.")
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
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.Option)
	if err != nil {
		return errors.Wrap(err, "get client from config")
	}
	resp, err := kargoSvcCli.ListWarehouses(
		ctx,
		connect.NewRequest(
			&v1alpha1.ListWarehousesRequest{
				Project: o.Project,
			},
		),
	)
	if err != nil {
		return errors.Wrap(err, "list warehouses")
	}

	res := make([]*kargoapi.Warehouse, 0, len(resp.Msg.GetWarehouses()))
	var resErr error
	if len(o.Names) == 0 {
		for _, w := range resp.Msg.GetWarehouses() {
			res = append(res, typesv1alpha1.FromWarehouseProto(w))
		}
	} else {
		warehousesByName :=
			make(map[string]*kargoapi.Warehouse, len(resp.Msg.GetWarehouses()))
		for _, w := range resp.Msg.GetWarehouses() {
			warehousesByName[w.GetMetadata().GetName()] =
				typesv1alpha1.FromWarehouseProto(w)
		}
		for _, name := range o.Names {
			if warehouse, ok := warehousesByName[name]; ok {
				res = append(res, warehouse)
			} else {
				resErr =
					goerrors.Join(err, errors.Errorf("warehouse %q not found", name))
			}
		}
	}
	if err := printObjects(o.Option, res); err != nil {
		return err
	}
	return resErr
}
