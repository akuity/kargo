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

func newGetStagesCommand(opt *option.Option) *cobra.Command {
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
				respProj, err := kargoSvcCli.ListProjects(ctx, connect.NewRequest(&v1alpha1.ListProjectsRequest{}))
				if err != nil {
					return errors.Wrap(err, "list projects")
				}
				for _, p := range respProj.Msg.GetProjects() {
					allProjects = append(allProjects, p.Name)
				}
			} else {
				allProjects = append(allProjects, project)
			}

			var allStages []*kargoapi.Stage

			// get all stages in project/all projects into a big slice
			for _, p := range allProjects {
				resp, err := kargoSvcCli.ListStages(ctx, connect.NewRequest(&v1alpha1.ListStagesRequest{
					Project: p,
				}))
				if err != nil {
					return errors.Wrap(err, "list stages")
				}
				for _, s := range resp.Msg.GetStages() {
					allStages = append(allStages, typesv1alpha1.FromStageProto(s))
				}
			}

			names := slices.Compact(args)

			var resErr error
			// if stage names were provided in cli - remove unneeded stages from the big slice
			if len(names) > 0 {
				i := 0
				for _, x := range allStages {
					if slices.Contains(names, x.Name) {
						allStages[i] = x
						i++
					}
				}
				// Prevent memory leak by erasing truncated pointers
				for j := i; j < len(allStages); j++ {
					allStages[j] = nil
				}
				allStages = allStages[:i]
			}
			if len(allStages) == 0 {
				resErr = goerrors.Join(err, errors.Errorf("No stages found"))
			} else {
				if err := printObjects(opt, allStages); err != nil {
					return err
				}
			}
			return resErr
		},
	}
	option.OptionalProject(opt.Project)(cmd.Flags())
	option.AllProjects(&opt.AllProjects)(cmd.Flags())
	opt.PrintFlags.AddFlags(cmd)
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
				duration.HumanDuration(time.Since(stage.CreationTimestamp.Time)),
				stage.Namespace,
			},
			Object: list.Items[i],
		}
	}
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Current Freight", Type: "string"},
			{Name: "Health", Type: "string"},
			{Name: "Age", Type: "string"},
			{Name: "Project", Type: "string"},
		},
		Rows: rows,
	}
}
