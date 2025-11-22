package delete

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
)

type deleteServiceAccountTokenOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	SystemLevel bool
	Project     string
	Names       []string
}

func newServiceAccountTokenCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &deleteServiceAccountTokenOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("deleted").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "serviceaccounttoken [--project=project] (NAME ...)",
		Aliases: []string{"serviceaccounttokens", "satoken", "satokens", "sat", "sats"},
		Short:   "Delete service account tokens by name",
		Args:    option.MinimumNArgs(1),
		Example: templates.Example(`
# Delete a service account token in my-project
kargo delete serviceaccounttoken --project=my-project my-token

# Delete multiple service account tokens
kargo delete serviceaccounttoken --project=my-project my-token1 my-token2

# Delete a service account token in the default project
kargo config set-project my-project
kargo delete serviceaccounttoken my-token

# Delete a system-level service account token
kargo delete serviceaccounttoken --system my-token
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

// addFlags adds the flags for the delete service account token options to the
// provided command.
func (o *deleteServiceAccountTokenOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(cmd.Flags(), &o.Project, o.Config.Project,
		"The Project for which to delete service account tokens. If not set, the "+
			"default project will be used.")
	option.System(
		cmd.Flags(), &o.SystemLevel, false,
		"Whether to delete system-level service account tokens instead of "+
			"project-level service account tokens.",
	)
	// project and system flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SystemFlag)
}

// complete sets the options from the command arguments.
func (o *deleteServiceAccountTokenOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *deleteServiceAccountTokenOptions) validate() error {
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

// run removes the service account token(s) from the project based on the
// options.
func (o *deleteServiceAccountTokenOptions) run(ctx context.Context) error {
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
		if _, err := kargoSvcCli.DeleteServiceAccountToken(
			ctx,
			connect.NewRequest(&v1alpha1.DeleteServiceAccountTokenRequest{
				SystemLevel: o.SystemLevel,
				Project:     o.Project,
				Name:        name,
			}),
		); err != nil {
			errs = append(errs, err)
			continue
		}
		_ = printer.PrintObj(&kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: o.Project,
			},
		}, o.Out)
	}
	return errors.Join(errs...)
}
