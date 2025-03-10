package get

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/io"
	"github.com/akuity/kargo/internal/cli/kubernetes"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
)

type getStagesOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	*getOptions

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
	Names   []string
}

func newGetStagesCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
	getOptions *getOptions,
) *cobra.Command {
	cmdOpts := &getStagesOptions{
		Config:     cfg,
		IOStreams:  streams,
		getOptions: getOptions,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "stages [--project=project] [NAME ...] [--no-headers]",
		Aliases: []string{"stage"},
		Short:   "Display one or many stages",
		Example: templates.Example(`
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
`),
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

	// Set the input/output streams for the command.
	io.SetIOStreams(cmd, cmdOpts.IOStreams)

	return cmd
}

// addFlags adds the flags for the get stages options to the provided command.
func (o *getStagesOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.PrintFlags.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
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
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
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
			return fmt.Errorf("list stages: %w", err)
		}
		return printObjects(resp.Msg.GetStages(), o.PrintFlags, o.IOStreams, o.NoHeaders)
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
		res = append(res, resp.Msg.GetStage())
	}

	if err = printObjects(res, o.PrintFlags, o.IOStreams, o.NoHeaders); err != nil {
		return fmt.Errorf("print stages: %w", err)
	}
	return errors.Join(errs...)
}

func newStageTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		stage := item.Object.(*kargoapi.Stage) // nolint: forcetypeassert
		var health string
		if stage.Status.Health != nil {
			health = string(stage.Status.Health.Status)
		}
		rows[i] = metav1.TableRow{
			Cells: []any{
				stage.Name,
				stage.Spec.Shard,
				stage.Status.FreightSummary,
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
			{Name: "Shard", Type: "string"},
			{Name: "Current Freight", Type: "string"},
			{Name: "Health", Type: "string"},
			{Name: "Phase", Type: "string"},
			{Name: "Age", Type: "string"},
		},
		Rows: rows,
	}
}
