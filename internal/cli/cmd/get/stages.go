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

type getStagesOptions struct {
	*option.Option
	Config config.CLIConfig

	Names []string
}

func newGetStagesCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmdOpts := &getStagesOptions{
		Option: opt,
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use:     "stages [--project=project] [NAME ...]",
		Aliases: []string{"stage"},
		Short:   "Display one or many stages",
		Example: `
# List all stages in my-project
kargo get stages --project=my-project

# List all stages in my-project in JSON output format
kargo get stages --project=my-project -o json

# Get the QA stage in my-project
kargo get stage --project=my-project qa

# List all stages in the default project
kargo config set-project my-project
kargo get stages

# Get a the QA stage in the default project
kargo config set-project my-project
kargo get stage qa
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

// addFlags adds the flags for the get stages options to the provided command.
func (o *getStagesOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Project,
		"The project for which to list stages. If not set, the default project will be used.",
	)
}

// complete sets the options from the command arguments.
func (o *getStagesOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *getStagesOptions) validate() error {
	if o.Project == "" {
		return errors.New("project is required")
	}
	return nil
}

// run gets the stages from the server and prints them to the console.
func (o *getStagesOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.Option)
	if err != nil {
		return errors.Wrap(err, "get client from config")
	}

	if len(o.Names) == 0 {

		var resp *connect.Response[v1alpha1.ListStagesResponse]
		if resp, err = kargoSvcCli.ListStages(
			ctx,
			connect.NewRequest(
				&v1alpha1.ListStagesRequest{
					Project: o.Project,
				},
			),
		); err != nil {
			return errors.Wrap(err, "list stages")
		}
		res := make([]*kargoapi.Stage, 0, len(resp.Msg.GetStages()))
		for _, stage := range resp.Msg.GetStages() {
			res = append(res, typesv1alpha1.FromStageProto(stage))
		}
		return printObjects(o.Option, res)

	}

	res := make([]*kargoapi.Stage, 0, len(o.Names))
	errs := make([]error, 0, len(o.Names))
	for _, name := range o.Names {
		var resp *connect.Response[v1alpha1.GetStageResponse]
		if resp, err = kargoSvcCli.GetStage(
			ctx,
			connect.NewRequest(
				&v1alpha1.GetStageRequest{
					Project: o.Project,
					Name:    name,
				},
			),
		); err != nil {
			errs = append(errs, err)
			continue
		}
		res = append(res, typesv1alpha1.FromStageProto(resp.Msg.GetStage()))
	}

	if err = printObjects(o.Option, res); err != nil {
		return errors.Wrap(err, "print stages")
	}

	if len(errs) == 0 {
		return nil
	}

	return goerrors.Join(errs...)
}

func newStageTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		stage := item.Object.(*kargoapi.Stage) // nolint: forcetypeassert
		var currentFreightID string
		if stage.Status.CurrentFreight != nil {
			currentFreightID = stage.Status.CurrentFreight.ID
		}
		var health string
		if stage.Status.Health != nil {
			health = string(stage.Status.Health.Status)
		}
		rows[i] = metav1.TableRow{
			Cells: []any{
				stage.Name,
				currentFreightID,
				health,
				stage.Status.Phase,
				duration.HumanDuration(time.Since(stage.CreationTimestamp.Time)),
			},
			Object: list.Items[i],
		}
	}
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Current Freight", Type: "string"},
			{Name: "Health", Type: "string"},
			{Name: "Phase", Type: "string"},
			{Name: "Age", Type: "string"},
		},
		Rows: rows,
	}
}
