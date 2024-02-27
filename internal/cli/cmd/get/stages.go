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

type getStagesOptions struct {
	*option.Option
}

// addFlags adds the flags for the get stages options to the provided command.
func (o *getStagesOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)

	option.Project(cmd.Flags(), &o.Project, o.Project,
		"The Project for which to list Stages. If not set, the default project will be used.")
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *getStagesOptions) validate() error {
	if o.Project == "" {
		return errors.New("project is required")
	}
	return nil
}

func newGetStagesCommand(
	cfg config.CLIConfig,
	opt *option.Option,
) *cobra.Command {
	cmdOpts := &getStagesOptions{Option: opt}

	cmd := &cobra.Command{
		Use:     "stages --project=project [NAME...]",
		Aliases: []string{"stage"},
		Short:   "Display one or many stages",
		Example: `
# List all stages in the project
kargo get stages --project=my-project

# List all stages in JSON output format
kargo get stages --project=my-project -o json

# Get a stage in the project
kargo get stages --project=my-project my-stage
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
			resp, err := kargoSvcCli.ListStages(ctx, connect.NewRequest(&v1alpha1.ListStagesRequest{
				Project: cmdOpts.Project,
			}))
			if err != nil {
				return errors.Wrap(err, "list stages")
			}

			names := slices.Compact(args)
			res := make([]*kargoapi.Stage, 0, len(resp.Msg.GetStages()))
			var resErr error
			if len(names) == 0 {
				for _, s := range resp.Msg.GetStages() {
					res = append(res, typesv1alpha1.FromStageProto(s))
				}
			} else {
				stagesByName := make(map[string]*kargoapi.Stage, len(resp.Msg.GetStages()))
				for _, s := range resp.Msg.GetStages() {
					stagesByName[s.GetMetadata().GetName()] = typesv1alpha1.FromStageProto(s)
				}
				for _, name := range names {
					if stage, ok := stagesByName[name]; ok {
						res = append(res, stage)
					} else {
						resErr = goerrors.Join(err, errors.Errorf("stage %q not found", name))
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
