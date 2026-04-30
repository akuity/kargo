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
	"github.com/akuity/kargo/pkg/client/generated/rbac"
	"github.com/akuity/kargo/pkg/client/generated/system"
)

type deleteTokenOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	SystemLevel bool
	Project     string
	Names       []string
}

func newTokenCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
) *cobra.Command {
	cmdOpts := &deleteTokenOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("deleted").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "token [--project=project] (NAME ...)",
		Aliases: []string{"tokens"},
		Short:   "Delete API tokens by name",
		Args:    option.MinimumNArgs(1),
		Example: templates.Example(`
# Delete an API token in my-project
kargo delete token --project=my-project my-token

# Delete multiple API tokens
kargo delete token --project=my-project my-token1 my-token2

# Delete an API token in the default project
kargo config set-project my-project
kargo delete token my-token

# Delete a system-level API token
kargo delete token --system my-token
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

// addFlags adds the flags for the delete API token options to the
// provided command.
func (o *deleteTokenOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(cmd.Flags(), &o.Project, o.Config.Project,
		"The Project for which to delete API tokens. If not set, the default "+
			"project will be used.")
	option.System(
		cmd.Flags(), &o.SystemLevel, false,
		"Whether to delete system-level API tokens instead of project-level API "+
			"tokens.",
	)
	// project and system flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SystemFlag)
}

// complete sets the options from the command arguments.
func (o *deleteTokenOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *deleteTokenOptions) validate() error {
	var errs []error
	if o.Project == "" && !o.SystemLevel {
		errs = append(errs, fmt.Errorf(
			"either %s or %s is required", option.ProjectFlag, option.SystemFlag,
		))
	}
	if len(o.Names) == 0 {
		errs = append(errs, errors.New("at least one argument is required"))
	}
	return errors.Join(errs...)
}

// run removes the API token(s) from the project based on the options.
func (o *deleteTokenOptions) run(ctx context.Context) error {
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

		if o.SystemLevel {
			if _, err := apiClient.Rbac.DeleteSystemAPIToken(
				rbac.NewDeleteSystemAPITokenParams().WithApitoken(name),
				nil,
			); err != nil {
				errs = append(errs, err)
				continue
			}
			namespace = systemConfig.SystemResourcesNamespace
		} else {
			if _, err := apiClient.Rbac.DeleteProjectAPIToken(
				rbac.NewDeleteProjectAPITokenParams().
					WithProject(o.Project).
					WithApitoken(name),
				nil,
			); err != nil {
				errs = append(errs, err)
				continue
			}
			namespace = o.Project
		}

		_ = printer.PrintObj(&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}, o.Out)
	}
	return errors.Join(errs...)
}
