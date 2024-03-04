package get

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
)

func NewCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get TYPE [NAME ...]",
		Short: "Display one or many resources",
		Args:  option.NoArgs,
		Example: `
# List all projects
kargo get projects

# List all stages in the project
kargo get stages --project=my-project

# List all promotions for the given stage
kargo get promotions --project=my-project --stage=my-stage
`,
	}

	// TODO: Factor out server flags to a higher level (root?) as they are
	//   common to almost all commands.
	option.InsecureTLS(cmd.PersistentFlags(), opt)
	option.LocalServer(cmd.PersistentFlags(), opt)

	// Register subcommands.
	cmd.AddCommand(newGetFreightCommand(cfg, opt))
	cmd.AddCommand(newGetProjectsCommand(cfg, opt))
	cmd.AddCommand(newGetPromotionsCommand(cfg, opt))
	cmd.AddCommand(newGetStagesCommand(cfg, opt))
	cmd.AddCommand(newGetWarehousesCommand(cfg, opt))
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

	if ptr.Deref(opt.PrintFlags.OutputFormat, "") != "" {
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
	var printObj runtime.Object
	switch any(t).(type) {
	case *kargoapi.Freight:
		printObj = newFreightTable(list)
	case *kargoapi.Project:
		printObj = newProjectTable(list)
	case *kargoapi.Promotion:
		printObj = newPromotionTable(list)
	case *kargoapi.Stage:
		printObj = newStageTable(list)
	default:
		printObj = list
	}
	return printers.NewTablePrinter(printers.PrintOptions{}).PrintObj(printObj, opt.IOStreams.Out)
}
