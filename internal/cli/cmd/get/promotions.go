package get

import (
	goerrors "errors"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type getPromotionsOptions struct {
	*option.Option

	Stage string
}

// addFlags adds the flags for the get promotions options to the provided command.
func (o *getPromotionsOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)

	option.Project(cmd.Flags(), &o.Project, o.Project,
		"The Project for which to list Promotions. If not set, the default project will be used.")
	option.Stage(cmd.Flags(), &o.Stage,
		"The Stage for which to list Promotions. If not set, all stages will be listed.")
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *getPromotionsOptions) validate() error {
	if o.Project == "" {
		return errors.New("project is required")
	}
	return nil
}

func newGetPromotionsCommand(
	cfg config.CLIConfig,
	opt *option.Option,
) *cobra.Command {
	cmdOpts := &getPromotionsOptions{Option: opt}

	cmd := &cobra.Command{
		Use:     "promotions --project=project [--stage=stage] [NAME...]",
		Aliases: []string{"promotion", "promos", "promo"},
		Short:   "Display one or many promotions",
		Example: `
# List all promotions in the project
kargo get promotions --project=my-project

# List all promotions in JSON output format
kargo get promotions --project=my-project -o json

# List all promotions for the stage
kargo get promotions --project=my-project --stage=my-stage

# Get a promotion in the project
kargo get promotions --project=my-project some-promotion
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if err := cmdOpts.validate(); err != nil {
				return err
			}

			req := &v1alpha1.ListPromotionsRequest{
				Project: cmdOpts.Project,
			}
			if cmdOpts.Stage != "" {
				req.Stage = proto.String(cmdOpts.Stage)
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, cmdOpts.Option)
			if err != nil {
				return errors.Wrap(err, "get client from config")
			}
			resp, err := kargoSvcCli.ListPromotions(ctx, connect.NewRequest(req))
			if err != nil {
				return errors.Wrap(err, "list promotions")
			}

			names := slices.Compact(args)
			res := make([]*kargoapi.Promotion, 0, len(resp.Msg.GetPromotions()))
			var resErr error
			if len(names) == 0 {
				for _, p := range resp.Msg.GetPromotions() {
					res = append(res, typesv1alpha1.FromPromotionProto(p))
				}
			} else {
				promotionsByName := make(map[string]*kargoapi.Promotion, len(resp.Msg.GetPromotions()))
				for _, p := range resp.Msg.GetPromotions() {
					promotionsByName[p.GetMetadata().GetName()] = typesv1alpha1.FromPromotionProto(p)
				}
				for _, name := range names {
					if promo, ok := promotionsByName[name]; ok {
						res = append(res, promo)
					} else {
						resErr = goerrors.Join(err, errors.Errorf("promotion %q not found", name))
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

func newPromotionTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		promo := item.Object.(*kargoapi.Promotion) // nolint: forcetypeassert
		rows[i] = metav1.TableRow{
			Cells: []any{
				promo.GetName(),
				promo.Spec.Stage,
				promo.Spec.Freight,
				promo.GetStatus().Phase,
				duration.HumanDuration(time.Since(promo.CreationTimestamp.Time)),
			},
			Object: list.Items[i],
		}
	}
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Stage", Type: "string"},
			{Name: "Freight", Type: "string"},
			{Name: "Phase", Type: "string"},
			{Name: "Age", Type: "string"},
		},
		Rows: rows,
	}
}
