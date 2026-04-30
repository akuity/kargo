package get

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/core"
)

type getConfigMapsOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	*getOptions

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
	Shared  bool
	System  bool
	Names   []string
}

func newGetConfigMapsCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
	getOptions *getOptions,
) *cobra.Command {
	cmdOpts := &getConfigMapsOptions{
		Config:     cfg,
		IOStreams:  streams,
		getOptions: getOptions,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "configmaps [--project=project | --shared | --system] [NAME ...] [--no-headers]",
		Aliases: []string{"configmap", "cm"},
		Short:   "Display one or many ConfigMaps",
		Example: templates.Example(`
# List all ConfigMaps in my-project
kargo get configmaps --project=my-project

# Get a specific ConfigMap in my-project
kargo get configmaps --project=my-project my-configmap

# List all ConfigMaps in the default project
kargo config set-project my-project
kargo get configmaps

# Get a specific ConfigMap in the default project
kargo config set-project my-project
kargo get configmaps my-configmap

# List shared ConfigMaps
kargo get configmaps --shared

# Get a specific shared ConfigMap
kargo get configmaps --shared my-configmap

# List system ConfigMaps
kargo get configmaps --system

# Get a specific system ConfigMap
kargo get configmaps --system my-configmap
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

// addFlags adds the flags for the get configmaps options to the provided command.
func (o *getConfigMapsOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project for which to list ConfigMaps. If not set, the default project will be used.",
	)
	option.Shared(
		cmd.Flags(), &o.Shared, false,
		"Whether to list shared ConfigMaps instead of project-specific ConfigMaps.",
	)
	option.System(
		cmd.Flags(), &o.System, false,
		"Whether to list system ConfigMaps instead of project-specific ConfigMaps.",
	)
	// project, shared, and system flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SharedFlag, option.SystemFlag)
}

// complete sets the options from the command arguments.
func (o *getConfigMapsOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *getConfigMapsOptions) validate() error {
	var errs []error
	if o.Project == "" && !o.Shared && !o.System {
		errs = append(errs, fmt.Errorf(
			"one of %s, %s, or %s is required",
			option.ProjectFlag, option.SharedFlag, option.SystemFlag,
		))
	}
	return errors.Join(errs...)
}

// run gets the ConfigMaps from the server and prints them to the console.
func (o *getConfigMapsOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	if len(o.Names) == 0 {
		var payload any

		switch {
		case o.System:
			var res *core.ListSystemConfigMapsOK
			if res, err = apiClient.Core.ListSystemConfigMaps(
				core.NewListSystemConfigMapsParams(),
				nil,
			); err != nil {
				return fmt.Errorf("list system ConfigMaps: %w", err)
			}
			payload = res.Payload
		case o.Shared:
			var res *core.ListSharedConfigMapsOK
			if res, err = apiClient.Core.ListSharedConfigMaps(
				core.NewListSharedConfigMapsParams(),
				nil,
			); err != nil {
				return fmt.Errorf("list shared ConfigMaps: %w", err)
			}
			payload = res.Payload
		default:
			var res *core.ListProjectConfigMapsOK
			if res, err = apiClient.Core.ListProjectConfigMaps(
				core.NewListProjectConfigMapsParams().WithProject(o.Project),
				nil,
			); err != nil {
				return fmt.Errorf("list project ConfigMaps: %w", err)
			}
			payload = res.Payload
		}

		var configMapsJSON []byte
		if configMapsJSON, err = json.Marshal(payload); err != nil {
			return err
		}
		configMaps := struct {
			Items []*corev1.ConfigMap `json:"items"`
		}{}
		if err = json.Unmarshal(configMapsJSON, &configMaps); err != nil {
			return err
		}
		return PrintConfigMaps(configMaps.Items, o.PrintFlags, o.IOStreams, o.NoHeaders)
	}

	res := make([]*corev1.ConfigMap, 0, len(o.Names))
	errs := make([]error, 0, len(o.Names))

	for _, name := range o.Names {
		var payload any

		switch {
		case o.System:
			var res *core.GetSystemConfigMapOK
			if res, err = apiClient.Core.GetSystemConfigMap(
				core.NewGetSystemConfigMapParams().WithConfigmap(name),
				nil,
			); err != nil {
				errs = append(errs, err)
				continue
			}
			payload = res.Payload
		case o.Shared:
			var res *core.GetSharedConfigMapOK
			if res, err = apiClient.Core.GetSharedConfigMap(
				core.NewGetSharedConfigMapParams().WithConfigmap(name),
				nil,
			); err != nil {
				errs = append(errs, err)
				continue
			}
			payload = res.Payload
		default:
			var res *core.GetProjectConfigMapOK
			if res, err = apiClient.Core.GetProjectConfigMap(
				core.NewGetProjectConfigMapParams().
					WithProject(o.Project).
					WithConfigmap(name),
				nil,
			); err != nil {
				errs = append(errs, err)
				continue
			}
			payload = res.Payload
		}

		var configMapJSON []byte
		if configMapJSON, err = json.Marshal(payload); err != nil {
			errs = append(errs, err)
			continue
		}
		var configMap *corev1.ConfigMap
		if err = json.Unmarshal(configMapJSON, &configMap); err != nil {
			errs = append(errs, err)
			continue
		}
		res = append(res, configMap)
	}

	if err = PrintConfigMaps(res, o.PrintFlags, o.IOStreams, o.NoHeaders); err != nil {
		return fmt.Errorf("print ConfigMaps: %w", err)
	}
	return errors.Join(errs...)
}

// PrintConfigMaps prints ConfigMaps to the output stream.
func PrintConfigMaps(
	configMaps []*corev1.ConfigMap,
	flags *genericclioptions.PrintFlags,
	streams genericiooptions.IOStreams,
	noHeaders bool,
) error {
	return PrintObjects(configMaps, flags, streams, noHeaders)
}

func newConfigMapsTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		configMap := item.Object.(*corev1.ConfigMap) // nolint: forcetypeassert
		rows[i] = metav1.TableRow{
			Cells: []any{
				configMap.Name,
				configMap.Annotations["kargo.akuity.io/description"],
				len(configMap.Data),
				duration.HumanDuration(time.Since(configMap.CreationTimestamp.Time)),
			},
			Object: list.Items[i],
		}
	}
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "Keys", Type: "integer"},
			{Name: "Age", Type: "string"},
		},
		Rows: rows,
	}
}
