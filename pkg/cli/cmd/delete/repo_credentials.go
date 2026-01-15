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
	"github.com/akuity/kargo/pkg/client/generated/credentials"
	"github.com/akuity/kargo/pkg/client/generated/system"
)

type deleteRepoCredentialsOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Shared  bool
	Project string
	Names   []string
}

func newRepoCredentialsCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &deleteRepoCredentialsOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("deleted").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use: "repo-credentials [--project=project] (NAME ...)",
		Aliases: []string{
			"repo-credential",
			"repo-creds",
			"repo-cred",
			"repocredentials",
			"repocredential",
			"repocreds",
			"repocred",
		},
		Short: "Delete repository credentials by name",
		Args:  cobra.MinimumNArgs(1),
		Example: templates.Example(`
# Delete repository credentials
kargo delete repo-credentials --project=my-project my-credentials

# Delete multiple repository credentials
kargo delete repo-credentials --project=my-project my-credentials1 my-credentials2

# Delete repository credentials from default project
kargo config set-project my-project
kargo delete repo-credentials my-credentials

# Delete shared repository credentials
kargo delete repo-credentials --shared my-credentials
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

// addFlags adds the flags for the delete repo-credentials options to the provided
// command.
func (o *deleteRepoCredentialsOptions) addFlags(cmd *cobra.Command) {
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
	// project and shared flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SharedFlag)
}

// complete sets the options from the command arguments.
func (o *deleteRepoCredentialsOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *deleteRepoCredentialsOptions) validate() error {
	var errs []error

	if o.Project == "" && !o.Shared {
		errs = append(errs, fmt.Errorf(
			"either %s or %s is required", option.ProjectFlag, option.SharedFlag,
		))
	}

	if len(o.Names) == 0 {
		errs = append(errs, errors.New("name is required"))
	}

	return errors.Join(errs...)
}

// run removes the credentials from the project based on the options.
func (o *deleteRepoCredentialsOptions) run(ctx context.Context) error {
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
		case o.Shared:
			if _, err := apiClient.Credentials.DeleteSharedRepoCredentials(
				credentials.NewDeleteSharedRepoCredentialsParams().
					WithRepoCredentials(name),
				nil,
			); err != nil {
				errs = append(errs, err)
				continue
			}
			namespace = systemConfig.SharedResourcesNamespace
		default:
			if _, err := apiClient.Credentials.DeleteProjectRepoCredentials(
				credentials.NewDeleteProjectRepoCredentialsParams().
					WithProject(o.Project).
					WithRepoCredentials(name),
				nil,
			); err != nil {
				errs = append(errs, err)
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
