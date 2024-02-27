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
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type getFreightOptions struct {
	*option.Option
}

// addFlags adds the flags for the get freight options to the provided command.
func (o *getFreightOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)

	option.Project(cmd.Flags(), &o.Project, o.Project,
		"The Project for which to list Freight. If not set, the default project will be used.")
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *getFreightOptions) validate() error {
	if o.Project == "" {
		return errors.New("project is required")
	}
	return nil
}

func newGetFreightCommand(
	cfg config.CLIConfig,
	opt *option.Option,
) *cobra.Command {
	cmdOpts := &getFreightOptions{Option: opt}

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

			if err := cmdOpts.validate(); err != nil {
				return err
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, cmdOpts.Option)
			if err != nil {
				return errors.Wrap(err, "get client from config")
			}
			resp, err := kargoSvcCli.QueryFreight(ctx, connect.NewRequest(&v1alpha1.QueryFreightRequest{
				Project: cmdOpts.Project,
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
			if err := printObjects(cmdOpts.Option, res); err != nil {
				return err
			}
			return resErr
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

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
