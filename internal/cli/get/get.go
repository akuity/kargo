package get

import (
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/utils/pointer"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/option"
)

func NewCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [--project=project] (RESOURCE) [NAME]...",
		Short: "Display one or many resources",
		Args:  cobra.MinimumNArgs(1),
		Example: `
# List all projects
kargo get projects

# List all stages in the project
kargo get --project= stages

# List all stages in JSON output format
kargo get stages my-project -o json
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			kargoSvcCli, err := client.GetClientFromConfig(ctx, opt)
			if err != nil {
				return errors.New("get client from config")
			}

			resource := strings.ToLower(strings.TrimSpace(args[0]))
			if resource == "" {
				return errors.New("resource is required")
			}

			var resErr error
			res := &metav1.List{
				TypeMeta: metav1.TypeMeta{
					APIVersion: metav1.Unversioned.String(),
					Kind:       "List",
				},
			}
			switch resource {
			case "project", "projects":
				names := slices.Compact(args[1:])
				filter, err := filterProjects(ctx, kargoSvcCli)
				if err != nil {
					return err
				}

				var projects []runtime.Object
				projects, resErr = filter(names...)
				res.Items = make([]runtime.RawExtension, 0, len(projects))
				for _, project := range projects {
					res.Items = append(res.Items, runtime.RawExtension{Object: project})
				}
				if len(names) == 1 {
					if len(res.Items) == 1 {
						_ = printResult(opt, res.Items[0].Object)
					}
					return resErr
				}
			case "stage", "stages":
				project := opt.Project.OrElse("")
				if project == "" {
					return errors.New("project is required")
				}
				names := slices.Compact(args[1:])
				filter, err := filterStages(ctx, kargoSvcCli, project)
				if err != nil {
					return err
				}

				var stages []runtime.Object
				stages, resErr = filter(names...)
				res.Items = make([]runtime.RawExtension, 0, len(stages))
				for _, stage := range stages {
					res.Items = append(res.Items, runtime.RawExtension{Object: stage})
				}
				if len(names) == 1 && len(res.Items) == 1 {
					_ = printStageResult(opt, res.Items[0].Object)
				} else {
					_ = printStageResult(opt, res)
				}
				return resErr
			default:
				return errors.Errorf("unknown resource %q", resource)
			}
			_ = printResult(opt, res)
			return resErr
		},
	}
	option.OptionalProject(opt.Project)(cmd.Flags())
	opt.PrintFlags.AddFlags(cmd)
	return cmd
}

func printStageResult(opt *option.Option, res runtime.Object) error {
	if pointer.StringDeref(opt.PrintFlags.OutputFormat, "") != "" {
		return printResult(opt, res)
	}
	var items []runtime.RawExtension
	if list, ok := res.(*metav1.List); ok {
		items = list.Items
	} else {
		items = []runtime.RawExtension{{Object: res}}
	}
	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Current Freight", Type: "string"},
			{Name: "Health", Type: "string"},
			{Name: "Age", Type: "string"},
		},
		Rows: make([]metav1.TableRow, len(items)),
	}
	for i, item := range items {
		// This func is only ever passed Stages
		stage := item.Object.(*kargoapi.Stage) // nolint: forcetypeassert
		var currentFreightID string
		if stage.Status.CurrentFreight != nil {
			currentFreightID = stage.Status.CurrentFreight.ID
		}
		var health string
		if stage.Status.Health != nil {
			health = string(stage.Status.Health.Status)
		}
		table.Rows[i] = metav1.TableRow{
			Cells: []any{
				stage.Name,
				currentFreightID,
				health,
				duration.HumanDuration(time.Since(stage.CreationTimestamp.Time)),
			},
			Object: item,
		}
	}
	return printers.NewTablePrinter(printers.PrintOptions{}).PrintObj(table, opt.IOStreams.Out)
}

func printResult(opt *option.Option, res runtime.Object) error {
	if pointer.StringDeref(opt.PrintFlags.OutputFormat, "") == "" {
		return printers.NewTablePrinter(printers.PrintOptions{}).PrintObj(res, opt.IOStreams.Out)
	}
	printer, err := opt.PrintFlags.ToPrinter()
	if err != nil {
		return errors.Wrap(err, "new printer")
	}
	return printer.PrintObj(res, opt.IOStreams.Out)
}
