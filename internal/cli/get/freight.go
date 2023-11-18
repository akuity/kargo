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

func newGetFreightCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "freight --project=project [NAME...]",
		Short: "Display one or many pieces of freight",
		Example: `
# List all freight in the project
kargo get freight --project=my-project

# List all freight in JSON output format
kargo get freight --project=my-project -o json

# Get a single piece of freight in the project
kargo get freight --project=my-project my-freight
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

			var allFreight []*kargoapi.Freight

			// get all freight in project/all projects into a big slice
			for _, p := range allProjects {
				resp, errF := kargoSvcCli.QueryFreight(ctx, connect.NewRequest(&v1alpha1.QueryFreightRequest{
					Project: p,
				}))
				if errF != nil {
					return errors.Wrap(errF, "list freight")
				}
				freight := resp.Msg.GetGroups()[""]
				for _, f := range freight.Freight {
					allFreight = append(allFreight, typesv1alpha1.FromFreightProto(f))
				}
			}

			names := slices.Compact(args)
			// We didn't specify any groupBy, so there should be one group with an
			// empty key

			var resErr error
			if len(names) > 0 {
				i := 0
				for _, x := range allFreight {
					if slices.Contains(names, x.Name) {
						allFreight[i] = x
						i++
					}
				}
				// Prevent memory leak by erasing truncated pointers
				for j := i; j < len(allFreight); j++ {
					allFreight[j] = nil
				}
				allFreight = allFreight[:i]
			}
			if len(allFreight) == 0 {
				resErr = goerrors.Join(err, errors.Errorf("No freight found"))
			} else {
				if errPr := printObjects(opt, allFreight); errPr != nil {
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
