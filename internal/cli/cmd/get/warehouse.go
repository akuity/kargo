package get

import (
	goerrors "errors"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func newGetWarehousesCommand(
	cfg config.CLIConfig,
	opt *option.Option,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "warehouses --project=project [NAME...]",
		Aliases: []string{"warehouse"},
		Short:   "Display one or many warehouses",
		Example: `
# List all warehouses in the project
kargo get warehouses --project=my-project

# List all warehouses in JSON output format
kargo get warehouses --project=my-project -o json

# Get a warehouse in the project
kargo get warehouses --project=my-project my-warehouse
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			project := opt.Project
			if project == "" {
				return errors.New("project is required")
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, opt)
			if err != nil {
				return errors.Wrap(err, "get client from config")
			}
			resp, err := kargoSvcCli.ListWarehouses(
				ctx,
				connect.NewRequest(
					&v1alpha1.ListWarehousesRequest{
						Project: project,
					},
				),
			)
			if err != nil {
				return errors.Wrap(err, "list warehouses")
			}

			names := slices.Compact(args)
			res := make([]*kargoapi.Warehouse, 0, len(resp.Msg.GetWarehouses()))
			var resErr error
			if len(names) == 0 {
				res = append(res, resp.Msg.GetWarehouses()...)
			} else {
				warehousesByName :=
					make(map[string]*kargoapi.Warehouse, len(resp.Msg.GetWarehouses()))
				for i := range resp.Msg.GetWarehouses() {
					w := resp.Msg.GetWarehouses()[i]
					warehousesByName[w.Name] = w
				}
				for _, name := range names {
					if warehouse, ok := warehousesByName[name]; ok {
						res = append(res, warehouse)
					} else {
						resErr =
							goerrors.Join(err, errors.Errorf("warehouse %q not found", name))
					}
				}
			}
			if err := printObjects(opt, res); err != nil {
				return err
			}
			return resErr
		},
	}
	option.Project(cmd.Flags(), opt, opt.Project)
	opt.PrintFlags.AddFlags(cmd)
	return cmd
}
