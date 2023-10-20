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
			if project == "" {
				return errors.New("project is required")
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, opt)
			if err != nil {
				return errors.New("get client from config")
			}
			resp, err := kargoSvcCli.QueryFreight(ctx, connect.NewRequest(&v1alpha1.QueryFreightRequest{
				Project: project,
			}))
			if err != nil {
				return errors.Wrap(err, "query freight")
			}

			names := slices.Compact(args)
			// We didn't specify any groupBy, so there should be one group with an
			// empty key
			freight := resp.Msg.GetGroups()[""]
			res := make([]*kargoapi.Freight, 0, len(freight.Freight))
			var resErr error
			if len(names) == 0 {
				for _, f := range freight.Freight {
					res = append(res, typesv1alpha1.FromFreightProto(f))
				}
			} else {
				freightByName := make(map[string]*kargoapi.Freight, len(freight.Freight))
				for _, f := range freight.Freight {
					freightByName[f.GetMetadata().GetName()] = typesv1alpha1.FromFreightProto(f)
				}
				for _, name := range names {
					if f, ok := freightByName[name]; ok {
						res = append(res, f)
					} else {
						resErr = goerrors.Join(err, errors.Errorf("freight %q not found", name))
					}
				}
			}
			if err := printObjects(opt, res); err != nil {
				return err
			}
			return resErr
		},
	}
	option.OptionalProject(opt.Project)(cmd.Flags())
	opt.PrintFlags.AddFlags(cmd)
	return cmd
}
