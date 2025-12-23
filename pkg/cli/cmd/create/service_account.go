package create

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	kargosvcapi "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
)

type createServiceAccountOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
	Name    string
}

func newServiceAccountCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &createServiceAccountOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("created").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "serviceaccount [--project=project] NAME",
		Aliases: []string{"sa"},
		Short:   "Create a service account",
		Args:    option.ExactArgs(1),
		Example: templates.Example(`
# Create a service account in a project
kargo create serviceaccount --project=my-project my-service-account

# Create a service account in the default project
kargo config set-project my-project
kargo create serviceaccount my-service-account
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

// addFlags adds the flags for the create service account options to the
// provided command.
func (o *createServiceAccountOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project in which to create the service account. If not set, the "+
			"default project will be used.",
	)
}

// complete sets the options from the command arguments.
func (o *createServiceAccountOptions) complete(args []string) {
	o.Name = strings.TrimSpace(args[0])
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *createServiceAccountOptions) validate() error {
	var errs []error
	if o.Project == "" {
		errs = append(errs, fmt.Errorf("%s is required", option.ProjectFlag))
	}
	if o.Name == "" {
		errs = append(errs, errors.New("one argument is required"))
	}
	return errors.Join(errs...)
}

// run creates a service account using the provided options.
func (o *createServiceAccountOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	resp, err := kargoSvcCli.CreateServiceAccount(
		ctx,
		connect.NewRequest(
			&kargosvcapi.CreateServiceAccountRequest{
				ServiceAccount: &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: o.Project,
						Name:      o.Name,
					},
				},
			},
		),
	)
	if err != nil {
		return fmt.Errorf("create service account: %w", err)
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("new printer: %w", err)
	}

	return printer.PrintObj(resp.Msg.ServiceAccount, o.Out)
}
