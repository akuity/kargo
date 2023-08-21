package create

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/utils/pointer"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/option"
)

func NewCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create (RESOURCE) (NAME)",
		Short: "Create a resource",
		Args:  cobra.MinimumNArgs(2),
		Example: `
# Create project
kargo create project my-project
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

			var res runtime.Object
			switch resource {
			case "project", "projects":
				name := args[1]
				res, err = createProject(ctx, kargoSvcCli, name)
				if err != nil {
					return errors.Wrap(err, "create project")
				}
			default:
				return errors.Errorf("unknown resource %q", resource)
			}

			if pointer.StringDeref(opt.PrintFlags.OutputFormat, "") == "" {
				_ = printers.NewTablePrinter(printers.PrintOptions{}).PrintObj(res, opt.IOStreams.Out)
				return nil
			}
			printer, err := opt.PrintFlags.ToPrinter()
			if err != nil {
				return errors.Wrap(err, "new printer")
			}
			return printer.PrintObj(res, opt.IOStreams.Out)
		},
	}
	opt.PrintFlags.AddFlags(cmd)
	return cmd
}
