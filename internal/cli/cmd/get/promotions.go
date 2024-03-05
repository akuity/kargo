package get

import (
	"context"
	goerrors "errors"
	"slices"
	"time"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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
	Config config.CLIConfig

	Project string
	Stage   string
	Names   []string
}

func newGetPromotionsCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmdOpts := &getPromotionsOptions{
		Option: opt,
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use:     "promotions [--project=project] [--stage=stage] [NAME ...]",
		Aliases: []string{"promotion", "promos", "promo"},
		Short:   "Display one or many promotions",
		Example: `
# List all promotions in my-project
kargo get promotions --project=my-project

# List all promotions in my-project in JSON output format
kargo get promotions --project=my-project -o json

# List all promotions for the QA stage in my-project
kargo get promotions --project=my-project --stage=qa

# Get a specific promotion in my-project
kargo get promotion --project=my-project abc1234

# List all promotions in the default project
kargo config set-project my-project
kargo get promotions

# List all promotions for the QA stage in the default project
kargo config set-project my-project
kargo get promotions --stage=qa

# Get a specific promotion in the default project
kargo config set-project my-project
kargo get promotion abc1234
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

// addFlags adds the flags for the get promotions options to the provided command.
func (o *getPromotionsOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project for which to list promotions. If not set, the default project will be used.",
	)
	option.Stage(
		cmd.Flags(), &o.Stage,
		"The stage for which to list promotions. If not set, all stages will be listed.",
	)
}

// complete sets the options from the command arguments.
func (o *getPromotionsOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *getPromotionsOptions) validate() error {
	if o.Project == "" {
		return errors.New("project is required")
	}
	return nil
}

// run gets the promotions from the server and prints them to the console.
func (o *getPromotionsOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.Option)
	if err != nil {
		return errors.Wrap(err, "get client from config")
	}

	if len(o.Names) == 0 {

		var resp *connect.Response[v1alpha1.ListPromotionsResponse]
		if resp, err = kargoSvcCli.ListPromotions(
			ctx,
			connect.NewRequest(
				&v1alpha1.ListPromotionsRequest{
					Project: o.Project,
					Stage:   &o.Stage,
				},
			),
		); err != nil {
			return errors.Wrap(err, "list promotions")
		}
		res := make([]*kargoapi.Promotion, 0, len(resp.Msg.GetPromotions()))
		for _, promotion := range resp.Msg.GetPromotions() {
			res = append(res, typesv1alpha1.FromPromotionProto(promotion))
		}
		return printObjects(o.Option, res)

	}

	res := make([]*kargoapi.Promotion, 0, len(o.Names))
	errs := make([]error, 0, len(o.Names))
	for _, name := range o.Names {
		var resp *connect.Response[v1alpha1.GetPromotionResponse]
		if resp, err = kargoSvcCli.GetPromotion(
			ctx,
			connect.NewRequest(
				&v1alpha1.GetPromotionRequest{
					Project: o.Project,
					Name:    name,
				},
			),
		); err != nil {
			errs = append(errs, err)
			continue
		}
		res = append(res, typesv1alpha1.FromPromotionProto(resp.Msg.GetPromotion()))
	}

	if err = printObjects(o.Option, res); err != nil {
		return errors.Wrap(err, "print promotions")
	}

	if len(errs) == 0 {
		return nil
	}

	return goerrors.Join(errs...)
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
