package delete

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
)

type deleteServiceAccountOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
	Names   []string
}

func newServiceAccountCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &deleteServiceAccountOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("deleted").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "serviceaccount [--project=project] (NAME ...)",
		Aliases: []string{"serviceaccounts", "sa", "sas"},
		Short:   "Delete service account by name",
		Args:    option.MinimumNArgs(1),
		Example: templates.Example(`
# Delete a service account
kargo delete serviceaccount --project=my-project my-service-account

# Delete multiple service accounts
kargo delete serviceaccount --project=my-project \
  my-service-account your-service-account

# Delete a service account in the default project
kargo config set-project my-project
kargo delete serviceaccount my-serviceaccount
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

// addFlags adds the flags for the delete service account options to the
// provided command.
func (o *deleteServiceAccountOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(cmd.Flags(), &o.Project, o.Config.Project,
		"The Project for which to delete service accounts. If not set, the "+
			"default project will be used.")
}

// complete sets the options from the command arguments.
func (o *deleteServiceAccountOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *deleteServiceAccountOptions) validate() error {
	var errs []error
	if o.Project == "" {
		errs = append(errs, fmt.Errorf("%s is required", option.ProjectFlag))
	}
	if len(o.Names) == 0 {
		errs = append(errs, errors.New("at least one argument is required"))
	}
	return errors.Join(errs...)
}

// run removes the service accounts(s) from the project based on the options.
func (o *deleteServiceAccountOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("create printer: %w", err)
	}

	var errs []error
	for _, name := range o.Names {
		if _, err := kargoSvcCli.DeleteServiceAccount(
			ctx,
			connect.NewRequest(&v1alpha1.DeleteServiceAccountRequest{
				Project: o.Project,
				Name:    name,
			}),
		); err != nil {
			errs = append(errs, err)
			continue
		}
		_ = printer.PrintObj(
			&corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: o.Project,
					Name:      name,
				},
			},
			o.Out,
		)
	}
	return errors.Join(errs...)
}
