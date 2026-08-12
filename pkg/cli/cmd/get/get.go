package get

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/cli-runtime/pkg/printers"
	sigyaml "sigs.k8s.io/yaml"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
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
	cmd.AddCommand(newGetConfigMapsCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newGetGenericCredentialsCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newGetRepoCredentialsCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newGetFreightCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newGetProjectConfigCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newGetProjectsCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newGetPromotionsCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newRolesCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newGetStagesCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newGetTokensCommand(cfg, streams, cmdOpts))
	cmd.AddCommand(newGetWarehousesCommand(cfg, streams, cmdOpts))

	return cmd
}

func (o *getOptions) addFlags(cmd *cobra.Command) {
	option.NoHeaders(cmd.PersistentFlags(), &o.NoHeaders)
}

func PrintObjects[T runtime.Object](
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
	case *corev1.ConfigMap:
		printObj = newConfigMapsTable(list)
	case *corev1.Secret:
		if len(list.Items) == 0 {
			return nil
		}
		// TODO(krancour): This is hacky and I don't love it
		secret := list.Items[0].Object.(*corev1.Secret) // nolint: forcetypeassert
		if secret.GetLabels()[rbacapi.LabelKeyAPIToken] == rbacapi.LabelValueTrue {
			printObj = newAPITokensTable(list)
		} else if secret.GetLabels()[kargoapi.LabelKeyCredentialType] == kargoapi.LabelValueCredentialTypeGeneric {
			printObj = newGenericCredentialsTable(list)
		} else if _, ok := secret.GetLabels()[kargoapi.LabelKeyCredentialType]; ok {
			printObj = newRepoCredentialsTable(list)
		} else {
			return nil
		}
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

// exportMetadataFields are ObjectMeta fields that are either server-managed
// or otherwise unsuitable to include in a manifest intended to be committed
// to git and reapplied with `kargo apply`.
var exportMetadataFields = []string{
	"creationTimestamp",
	"deletionGracePeriodSeconds",
	"deletionTimestamp",
	"generation",
	"managedFields",
	"resourceVersion",
	"selfLink",
	"uid",
}

// PrintExportableObjects behaves like PrintObjects, but additionally supports
// exporting objects as git-friendly manifests. When export is true, each
// object is sanitized (see sanitizeForExport) and printed as YAML regardless
// of any output format flag. If outputFile is non-empty, output is written
// there instead of streams.Out.
func PrintExportableObjects[T runtime.Object](
	objects []T,
	flags *genericclioptions.PrintFlags,
	streams genericiooptions.IOStreams,
	noHeaders bool,
	export bool,
	outputFile string,
) error {
	if !export {
		return PrintObjects(objects, flags, streams, noHeaders)
	}

	w := streams.Out
	if outputFile != "" {
		f, err := os.Create(outputFile)
		if err != nil {
			return fmt.Errorf("open output file: %w", err)
		}
		defer func() {
			_ = f.Close()
		}()
		w = f
	}

	printer := &printers.YAMLPrinter{}
	for _, obj := range objects {
		sanitized, err := sanitizeForExport(obj)
		if err != nil {
			return fmt.Errorf("sanitize object for export: %w", err)
		}
		if err := printer.PrintObj(sanitized, w); err != nil {
			return fmt.Errorf("print object: %w", err)
		}
	}
	return nil
}

// sanitizeForExport returns obj as an Unstructured object with its .status
// and non-applyable ObjectMeta fields (see exportMetadataFields) stripped, so
// that it is suitable for committing to git and reapplying with
// `kargo apply`.
func sanitizeForExport(obj runtime.Object) (*unstructured.Unstructured, error) {
	obj = obj.DeepCopyObject()
	if obj.GetObjectKind().GroupVersionKind().Empty() {
		gvks, _, err := kubernetes.GetScheme().ObjectKinds(obj)
		if err != nil {
			return nil, fmt.Errorf("look up group version kind: %w", err)
		}
		obj.GetObjectKind().SetGroupVersionKind(gvks[0])
	}

	data, err := sigyaml.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("marshal object: %w", err)
	}
	m := map[string]any{}
	if err := sigyaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("unmarshal object: %w", err)
	}

	delete(m, "status")
	if metadata, ok := m["metadata"].(map[string]any); ok {
		for _, field := range exportMetadataFields {
			delete(metadata, field)
		}
	}

	return &unstructured.Unstructured{Object: m}, nil
}
