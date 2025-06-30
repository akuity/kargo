package get

import (
	"fmt"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/printers"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
)

type getOptions struct {
	NoHeaders bool
}

func NewCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get TYPE [NAME ...]",
		Short: "Display one or many resources",
		Args:  option.NoArgs,
		Example: templates.Example(`
# List all projects
kargo get projects

# List all stages in the project
kargo get stages --project=my-project

# List all promotions for the given stage
kargo get promotions --project=my-project --stage=my-stage
`),
	}

	cmdOpts := &getOptions{}

	cmdOpts.addFlags(cmd)

	// Register subcommands.
	cmd.AddCommand(newGetClusterConfigCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newGetCredentialsCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newGetFreightCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newGetProjectConfigCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newGetProjectsCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newGetPromotionsCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newRolesCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newGetStagesCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newGetWarehousesCommand(cfg, streams, cmdOpts))

	return cmd
}

func (o *getOptions) addFlags(cmd *cobra.Command) {
	option.NoHeaders(cmd.PersistentFlags(), &o.NoHeaders)
}

func printObjects[T runtime.Object](
	objects []T,
	flags *genericclioptions.PrintFlags,
	streams genericiooptions.IOStreams,
	noHeaders bool,
) error {
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

	if flags.OutputFlagSpecified != nil && flags.OutputFlagSpecified() {
		printer, err := flags.ToPrinter()
		if err != nil {
			return fmt.Errorf("new printer: %w", err)
		}
		if len(list.Items) == 1 {
			return printer.PrintObj(list.Items[0].Object, streams.Out)
		}
		return printer.PrintObj(list, streams.Out)
	}

	var t T
	var printObj runtime.Object
	switch any(t).(type) {
	case *corev1.Secret:
		printObj = newCredentialsTable(list)
	case *kargoapi.ClusterConfig:
		printObj = newClusterConfigTable(list)
	case *kargoapi.Freight:
		printObj = newFreightTable(list)
	case *kargoapi.Project:
		printObj = newProjectTable(list)
	case *kargoapi.ProjectConfig:
		printObj = newProjectConfigTable(list)
	case *kargoapi.Promotion:
		printObj = newPromotionTable(list)
	case *rbacapi.Role:
		printObj = newRoleTable(list)
	case *rbacapi.RoleResources:
		printObj = newRoleResourcesTable(list)
	case *kargoapi.Stage:
		printObj = newStageTable(list)
	case *kargoapi.Warehouse:
		printObj = newWarehouseTable(list)
	default:
		printObj = list
	}
	return printers.
		NewTablePrinter(
			printers.PrintOptions{
				NoHeaders: noHeaders,
			},
		).
		PrintObj(printObj, streams.Out)
}
