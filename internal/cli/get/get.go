package get

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/utils/pointer"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/option"
)

func NewCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get (RESOURCE) [NAME]...",
		Short: "Display one or many resources",
		Example: `
# List all projects
kargo get projects

# List all stages in the project
kargo get stages --project=my-project

# List all promotions for the given stage
kargo get promotions --project=my-project --stage=my-stage
`,
	}
	// Subcommands
	cmd.AddCommand(newGetFreightCommand(opt))
	cmd.AddCommand(newGetProjectsCommand(opt))
	cmd.AddCommand(newGetPromotionsCommand(opt))
	cmd.AddCommand(newGetStagesCommand(opt))
	cmd.AddCommand(newGetWarehousesCommand(opt))
	return cmd
}

func printObjects[T runtime.Object](opt *option.Option, objects []T) error {
	items := make([]runtime.RawExtension, len(objects))
	for i, obj := range objects {
		items[i] = runtime.RawExtension{Object: obj}
	}
	list := &metav1.List{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metav1.Unversioned.String(),
			Kind:       "List",
		},
		Items: items,
	}

	if pointer.StringDeref(opt.PrintFlags.OutputFormat, "") != "" {
		printer, err := opt.PrintFlags.ToPrinter()
		if err != nil {
			return errors.Wrap(err, "new printer")
		}
		if len(list.Items) == 1 {
			return printer.PrintObj(list.Items[0].Object, opt.IOStreams.Out)
		}
		return printer.PrintObj(list, opt.IOStreams.Out)
	}

	var t T
	switch any(t).(type) {
	case *kargoapi.Stage:
		table := newStageTable(list)
		return printers.NewTablePrinter(printers.PrintOptions{}).PrintObj(table, opt.IOStreams.Out)
	case *kargoapi.Promotion:
		table := newPromotionTable(list)
		return printers.NewTablePrinter(printers.PrintOptions{}).PrintObj(table, opt.IOStreams.Out)
	default:
		return printers.NewTablePrinter(printers.PrintOptions{}).PrintObj(list, opt.IOStreams.Out)
	}
}
