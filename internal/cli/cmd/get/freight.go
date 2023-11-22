package get

import (
	goerrors "errors"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"

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

			project := opt.Project
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
				freightByAlias := make(map[string]*kargoapi.Freight, len(freight.Freight))
				for _, f := range freight.Freight {
					fr := typesv1alpha1.FromFreightProto(f)
					freightByName[f.GetMetadata().GetName()] = fr
					if f.GetMetadata().GetLabels() != nil {
						freightByAlias[f.GetMetadata().GetLabels()[kargoapi.AliasLabelKey]] = fr
					}
				}
				selectedFreight := make(map[string]struct{}, len(names))
				for _, name := range names {
					f, ok := freightByName[name]
					if !ok {
						f, ok = freightByAlias[name]
					}
					if ok {
						if _, selected := selectedFreight[f.Name]; !selected {
							res = append(res, f)
							selectedFreight[f.Name] = struct{}{}
						}
					} else {
						resErr =
							goerrors.Join(err, errors.Errorf("freight %q not found", name))
					}
				}
			}
			if err := printObjects(opt, res); err != nil {
				return err
			}
			return resErr
		},
	}
	option.Project(&opt.Project, opt.Project)(cmd.Flags())
	opt.PrintFlags.AddFlags(cmd)
	return cmd
}

func newFreightTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		freight := item.Object.(*kargoapi.Freight) // nolint: forcetypeassert
		var alias string
		if freight.Labels != nil {
			alias = freight.Labels[kargoapi.AliasLabelKey]
		}
		rows[i] = metav1.TableRow{
			Cells: []any{
				freight.Name,
				alias,
				duration.HumanDuration(time.Since(freight.CreationTimestamp.Time)),
			},
			Object: list.Items[i],
		}
	}
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name/ID", Type: "string"},
			{Name: "Alias", Type: "string"},
			{Name: "Age", Type: "string"},
		},
		Rows: rows,
	}
}
