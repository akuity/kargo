package get

import (
	"errors"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/utils/pointer"

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
				return pkgerrors.New("get client from config")
			}

			resource := strings.ToLower(strings.TrimSpace(args[0]))
			if resource == "" {
				return pkgerrors.New("resource is required")
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
				if len(names) == 1 {
					if len(res.Items) == 1 {
						_ = printResult(opt, res.Items[0].Object)
					}
					return resErr
				}
			default:
				return pkgerrors.Errorf("unknown resource %q", resource)
			}
			_ = printResult(opt, res)
			return resErr
		},
	}
	option.OptionalProject(opt.Project)(cmd.Flags())
	opt.PrintFlags.AddFlags(cmd)
	return cmd
}

func printResult(opt *option.Option, res runtime.Object) error {
	if pointer.StringDeref(opt.PrintFlags.OutputFormat, "") == "" {
		return printers.NewTablePrinter(printers.PrintOptions{}).PrintObj(res, opt.IOStreams.Out)
	}
	printer, err := opt.PrintFlags.ToPrinter()
	if err != nil {
		return pkgerrors.Wrap(err, "new printer")
	}
	return printer.PrintObj(res, opt.IOStreams.Out)
}
