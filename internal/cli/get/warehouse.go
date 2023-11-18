package get

import (
	goerrors "errors"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func newGetWarehousesCommand(opt *option.Option) *cobra.Command {
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

			project := opt.Project.OrElse("")
			if project == "" && !opt.AllProjects {
				return errors.New("project or all-projects is required")
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, opt)
			if err != nil {
				return errors.New("get client from config")
			}

			var allProjects []string

			if opt.AllProjects {
				respProj, errP := kargoSvcCli.ListProjects(ctx, connect.NewRequest(&v1alpha1.ListProjectsRequest{}))
				if errP != nil {
					return errors.Wrap(errP, "list projects")
				}
				for _, p := range respProj.Msg.GetProjects() {
					allProjects = append(allProjects, p.Name)
				}
			} else {
				allProjects = append(allProjects, project)
			}

			var allWarehouses []*kargoapi.Warehouse

			// get all warehouses in project/all projects into a big slice
			for _, p := range allProjects {
				resp, errW := kargoSvcCli.ListWarehouses(ctx, connect.NewRequest(&v1alpha1.ListWarehousesRequest{
					Project: p,
				}))
				if errW != nil {
					return errors.Wrap(errW, "list warehouse")
				}

				for _, w := range resp.Msg.GetWarehouses() {
					allWarehouses = append(allWarehouses, typesv1alpha1.FromWarehouseProto(w))
				}
			}

			names := slices.Compact(args)

			var resErr error
			// if warehouse names were provided in cli - remove unneeded ones from the big slice
			if len(names) > 0 {
				i := 0
				for _, x := range allWarehouses {
					if slices.Contains(names, x.Name) {
						allWarehouses[i] = x
						i++
					}
				}
				// Prevent memory leak by erasing truncated pointers
				for j := i; j < len(allWarehouses); j++ {
					allWarehouses[j] = nil
				}
				allWarehouses = allWarehouses[:i]
			}
			if len(allWarehouses) == 0 {
				resErr = goerrors.Join(err, errors.Errorf("No warehouses found"))
			} else {
				if errPr := printObjects(opt, allWarehouses); errPr != nil {
					return errPr
				}
			}
			return resErr
		},
	}
	option.OptionalProject(opt.Project)(cmd.Flags())
	option.AllProjects(&opt.AllProjects)(cmd.Flags())
	opt.PrintFlags.AddFlags(cmd)
	return cmd
}
