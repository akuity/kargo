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

func newGetPromotionsCommand(
	cfg config.CLIConfig,
	opt *option.Option,
) *cobra.Command {
	var stage string
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

			project := opt.Project
			if project == "" {
				return errors.New("project is required")
			}
			req := &v1alpha1.ListPromotionsRequest{
				Project: project,
			}
			if stage != "" {
				req.Stage = proto.String(stage)
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, opt)
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
			if err := printObjects(opt, res); err != nil {
				return err
			}
			return resErr
		},
	}
	option.Project(cmd.Flags(), opt, opt.Project)
	option.Stage(cmd.Flags(), &stage)
	opt.PrintFlags.AddFlags(cmd)
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
