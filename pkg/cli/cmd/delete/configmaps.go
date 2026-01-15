package delete

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/core"
	"github.com/akuity/kargo/pkg/client/generated/system"
)

type deleteConfigMapOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
	Shared  bool
	System  bool
	Names   []string
}

func newConfigMapCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &deleteConfigMapOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("deleted").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "configmap [--project=project | --shared | --system] (NAME ...)",
		Aliases: []string{"configmaps", "cm"},
		Short:   "Delete ConfigMaps by name",
		Args:    cobra.MinimumNArgs(1),
		Example: templates.Example(`
# Delete a ConfigMap
kargo delete configmap --project=my-project my-configmap

# Delete multiple ConfigMaps
kargo delete configmap --project=my-project my-configmap1 my-configmap2

# Delete a ConfigMap from the default project
kargo config set-project my-project
kargo delete configmap my-configmap

# Delete a shared ConfigMap
kargo delete configmap --shared my-configmap

# Delete a system ConfigMap
kargo delete configmap --system my-configmap
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

// addFlags adds the flags for the delete configmap options to the provided command.
func (o *deleteConfigMapOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project for which to delete ConfigMaps. If not set, the default project will be used.",
	)
	option.Shared(
		cmd.Flags(), &o.Shared, false,
		"Whether to delete shared ConfigMaps instead of project-specific ConfigMaps.",
	)
	option.System(
		cmd.Flags(), &o.System, false,
		"Whether to delete system ConfigMaps.",
	)
	// project, shared, and system flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SharedFlag, option.SystemFlag)
}

// complete sets the options from the command arguments.
func (o *deleteConfigMapOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *deleteConfigMapOptions) validate() error {
	var errs []error

	if o.Project == "" && !o.Shared && !o.System {
		errs = append(errs, fmt.Errorf(
			"one of %s, %s, or %s is required",
			option.ProjectFlag, option.SharedFlag, option.SystemFlag,
		))
	}

	if len(o.Names) == 0 {
		errs = append(errs, errors.New("name is required"))
	}

	return errors.Join(errs...)
}

// run removes the ConfigMaps based on the options.
func (o *deleteConfigMapOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	res, err := apiClient.System.GetConfig(system.NewGetConfigParams(), nil)
	if err != nil {
		return fmt.Errorf("get system config: %w", err)
	}
	systemConfig := res.Payload

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("create printer: %w", err)
	}

	var errs []error
	for _, name := range o.Names {
		var namespace string

		switch {
		case o.System:
			if _, err := apiClient.Core.DeleteSystemConfigMap(
				core.NewDeleteSystemConfigMapParams().
					WithConfigmap(name),
				nil,
			); err != nil {
				errs = append(errs, err)
				continue
			}
			namespace = systemConfig.SystemResourcesNamespace
		case o.Shared:
			if _, err := apiClient.Core.DeleteSharedConfigMap(
				core.NewDeleteSharedConfigMapParams().
					WithConfigmap(name),
				nil,
			); err != nil {
				errs = append(errs, err)
				continue
			}
			namespace = systemConfig.SharedResourcesNamespace
		default:
			if _, err := apiClient.Core.DeleteProjectConfigMap(
				core.NewDeleteProjectConfigMapParams().
					WithProject(o.Project).
					WithConfigmap(name),
				nil,
			); err != nil {
				errs = append(errs, err)
				continue
			}
			namespace = o.Project
		}

		_ = printer.PrintObj(
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			},
			o.Out,
		)
	}
	return errors.Join(errs...)
}
