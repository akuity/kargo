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
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type PromotionsFlags struct {
	Stage option.Optional[string]
}

func newGetPromotionsCommand(opt *option.Option) *cobra.Command {
	flag := PromotionsFlags{
		Stage: option.OptionalString(),
	}
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

			var allPromotions []*kargoapi.Promotion

			// get all promotions in project/all projects into a big slice
			for _, p := range allProjects {

				req := &v1alpha1.ListPromotionsRequest{
					Project: p,
				}
				if stage, ok := flag.Stage.Get(); ok {
					req.Stage = proto.String(stage)
				}

				resp, errP := kargoSvcCli.ListPromotions(ctx, connect.NewRequest(req))
				if errP != nil {
					return errors.Wrap(errP, "list promotions")
				}
				for _, s := range resp.Msg.GetPromotions() {
					allPromotions = append(allPromotions, typesv1alpha1.FromPromotionProto(s))
				}
			}

			names := slices.Compact(args)

			var resErr error
			// if promotion names were provided in cli - remove unneeded ones from the big slice
			if len(names) > 0 {
				i := 0
				for _, x := range allPromotions {
					if slices.Contains(names, x.Name) {
						allPromotions[i] = x
						i++
					}
				}
				// Prevent memory leak by erasing truncated pointers
				for j := i; j < len(allPromotions); j++ {
					allPromotions[j] = nil
				}
				allPromotions = allPromotions[:i]
			}
			if len(allPromotions) == 0 {
				resErr = goerrors.Join(err, errors.Errorf("No promotions found"))
			} else {
				if errPr := printObjects(opt, allPromotions); errPr != nil {
					return errPr
				}
			}
			return resErr
		},
	}
	option.OptionalProject(opt.Project)(cmd.Flags())
	option.AllProjects(&opt.AllProjects)(cmd.Flags())
	option.OptionalStage(flag.Stage)(cmd.Flags())
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
				promo.Namespace,
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
			{Name: "Project", Type: "string"},
		},
		Rows: rows,
	}
}
