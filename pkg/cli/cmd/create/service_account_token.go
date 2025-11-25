package create

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
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

type createServiceAccountTokenOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	SystemLevel        bool
	Project            string
	ServiceAccountName string
	Name               string
}

func newServiceAccountTokenCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
) *cobra.Command {
	cmdOpts := &createServiceAccountTokenOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("created").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "serviceaccounttoken [--project=project] --service-account=service-account NAME",
		Aliases: []string{"satoken", "sat"},
		Short:   "Generate and retrieve a token for the specified service account",
		Args:    option.ExactArgs(1),
		Example: templates.Example(`
# Create a token for service account my-service-account in my-project
kargo create serviceaccounttoken --project=my-project \
  --service-account=my-service-account my-token

# Create a token for service account my-service-account in the default project
kargo config set-project my-project
kargo create serviceaccounttoken --service-account=my-service-account my-token

# Create a token for system-level service account kargo-admin
kargo create serviceaccounttoken --system --service-account=kargo-admin my-token
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

// addFlags adds the flags for the create service account token options to the
// provided command.
func (o *createServiceAccountTokenOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project in which to create a token. If not set, the default project "+
			"will be used.",
	)
	option.System(
		cmd.Flags(), &o.SystemLevel, false,
		"Whether to create a token for a system-level service account instead of "+
			"a project-level service account.",
	)
	// project and system flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SystemFlag)

	option.ServiceAccount(
		cmd.Flags(),
		&o.ServiceAccountName,
		"The service account for which to create a token.",
	)
	if err := cmd.MarkFlagRequired(option.ServiceAccountFlag); err != nil {
		panic(fmt.Errorf(
			"could not mark %s flag as required: %w", option.ServiceAccountFlag, err,
		))
	}
}

// complete sets the options from the command arguments.
func (o *createServiceAccountTokenOptions) complete(args []string) {
	o.Name = strings.TrimSpace(args[0])
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *createServiceAccountTokenOptions) validate() error {
	var errs []error
	if o.Project == "" && !o.SystemLevel {
		errs = append(errs, fmt.Errorf(
			"either %s or %s is required", option.ProjectFlag, option.SystemFlag,
		))
	}
	// This flag is marked as required, but a user could still have provide an
	// empty string as the flag's value.
	if o.ServiceAccountName == "" {
		errs = append(errs, fmt.Errorf("%s is required", option.ServiceAccountFlag))
	}
	return errors.Join(errs...)
}

// run creates a service account token and prints it to the console.
func (o *createServiceAccountTokenOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	resp, err := kargoSvcCli.CreateServiceAccountToken(
		ctx,
		connect.NewRequest(
			&v1alpha1.CreateServiceAccountTokenRequest{
				SystemLevel:        o.SystemLevel,
				Project:            o.Project,
				ServiceAccountName: o.ServiceAccountName,
				Name:               o.Name,
			},
		),
	)
	if err != nil {
		return fmt.Errorf("get service account token: %w", err)
	}

	// If user specified an output format (yaml, json, etc.), use it
	if o.OutputFlagSpecified != nil && o.OutputFlagSpecified() {
		printer, err := o.ToPrinter()
		if err != nil {
			return fmt.Errorf("new printer: %w", err)
		}
		return printer.PrintObj(resp.Msg.TokenSecret, o.Out)
	}

	// Otherwise, print the token value clearly so users don't miss it
	tokenValue := string(resp.Msg.TokenSecret.Data["token"])
	if tokenValue == "" {
		return fmt.Errorf("token data is empty")
	}

	fmt.Fprintf(o.Out, "Token created successfully!\n\n")
	fmt.Fprintf(o.Out, "IMPORTANT: Save this token securely. It will not be shown again.\n\n")
	fmt.Fprintf(o.Out, "Token: %s\n", tokenValue)

	return nil
}
