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
)

type deleteGenericCredentialsOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
	Shared  bool
	System  bool
	Names   []string
}

func newGenericCredentialsCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &deleteGenericCredentialsOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("deleted").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use: "generic-credentials [--project=project | --shared | --system] (NAME ...)",
		Aliases: []string{
			"generic-credential",
			"generic-creds",
			"generic-cred",
			"genericcredentials",
			"genericcredential",
			"genericcreds",
			"genericcred",
		},
		Short: "Delete generic credentials by name",
		Args:  cobra.MinimumNArgs(1),
		Example: templates.Example(`
# Delete generic credentials
kargo delete generic-credentials --project=my-project my-credentials

# Delete multiple generic credentials
kargo delete generic-credentials --project=my-project my-credentials1 my-credentials2

# Delete generic credentials from default project
kargo config set-project my-project
kargo delete generic-credentials my-credentials

# Delete shared generic credentials
kargo delete generic-credentials --shared my-credentials

# Delete system generic credentials
kargo delete generic-credentials --system my-credentials
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

// addFlags adds the flags for the delete generic-credentials options to the provided
// command.
func (o *deleteGenericCredentialsOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project for which to delete credentials. If not set, the default project will be used.",
	)
	option.Shared(
		cmd.Flags(), &o.Shared, false,
		"Whether to delete shared credentials that can be used across all projects.",
	)
	option.System(
		cmd.Flags(), &o.System, false,
		"Whether to delete system credentials.",
	)
	// project, shared, and system flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SharedFlag, option.SystemFlag)
}

// complete sets the options from the command arguments.
func (o *deleteGenericCredentialsOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *deleteGenericCredentialsOptions) validate() error {
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

// run removes the credentials from the project based on the options.
func (o *deleteGenericCredentialsOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	systemConfig, httpRes, err := apiClient.SystemAPI.GetConfig(ctx).Execute()
	if httpRes != nil {
		_ = httpRes.Body.Close()
	}
	if err != nil {
		return fmt.Errorf("get system config: %w", client.APIError(err))
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("create printer: %w", err)
	}

	var errs []error
	for _, name := range o.Names {
		var namespace string

		switch {
		case o.System:
			delRes, delErr := apiClient.CredentialsAPI.DeleteSystemGenericCredentials(ctx, name).Execute()
			if delRes != nil {
				_ = delRes.Body.Close()
			}
			if delErr != nil {
				errs = append(errs, client.APIError(delErr))
				continue
			}
			namespace = systemConfig.GetSystemResourcesNamespace()
		case o.Shared:
			delRes, delErr := apiClient.CredentialsAPI.DeleteSharedGenericCredentials(ctx, name).Execute()
			if delRes != nil {
				_ = delRes.Body.Close()
			}
			if delErr != nil {
				errs = append(errs, client.APIError(delErr))
				continue
			}
			namespace = systemConfig.GetSharedResourcesNamespace()
		default:
			delRes, delErr := apiClient.CredentialsAPI.DeleteProjectGenericCredentials(ctx, o.Project, name).Execute()
			if delRes != nil {
				_ = delRes.Body.Close()
			}
			if delErr != nil {
				errs = append(errs, client.APIError(delErr))
				continue
			}
			namespace = o.Project
		}

		_ = printer.PrintObj(
			&corev1.Secret{
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
