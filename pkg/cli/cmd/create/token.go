package create

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/models"
	"github.com/akuity/kargo/pkg/client/generated/rbac"
)

type createTokenOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	SystemLevel bool
	Project     string
	RoleName    string
	Name        string
}

func newTokenCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
) *cobra.Command {
	cmdOpts := &createTokenOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("created").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "token [--project=project] --role=role NAME",
		Aliases: []string{"token"},
		Short:   "Generate and retrieve a token for the specified role",
		Args:    option.ExactArgs(1),
		Example: templates.Example(`
# Create a token for role my-role in my-project
kargo create token --project=my-project --role=my-role my-token

# Create a token for role my-role in the default project
kargo config set-project my-project
kargo create token --role=my-role my-token

# Create a token for system-level role kargo-admin
kargo create token --system --role=kargo-admin my-token
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

// addFlags adds the flags for the create API token options to the provided
// command.
func (o *createTokenOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project in which to create a token. If not set, the default project "+
			"will be used.",
	)
	option.System(
		cmd.Flags(), &o.SystemLevel, false,
		"Whether to create a token for a system-level role instead of a "+
			"project-level role.",
	)
	// project and system flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SystemFlag)

	option.Role(cmd.Flags(), &o.RoleName, "The role for which to create a token.")
	if err := cmd.MarkFlagRequired(option.RoleFlag); err != nil {
		panic(fmt.Errorf(
			"could not mark %s flag as required: %w", option.RoleFlag, err,
		))
	}
}

// complete sets the options from the command arguments.
func (o *createTokenOptions) complete(args []string) {
	o.Name = strings.TrimSpace(args[0])
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *createTokenOptions) validate() error {
	var errs []error
	if o.Project == "" && !o.SystemLevel {
		errs = append(errs, fmt.Errorf(
			"either %s or %s is required", option.ProjectFlag, option.SystemFlag,
		))
	}
	// This flag is marked as required, but a user could still have provide an
	// empty string as the flag's value.
	if o.RoleName == "" {
		errs = append(errs, fmt.Errorf("%s is required", option.RoleFlag))
	}
	return errors.Join(errs...)
}

// run creates an API token and prints it to the console.
func (o *createTokenOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	var payload any
	if o.SystemLevel {
		var res *rbac.CreateSystemAPITokenCreated
		if res, err = apiClient.Rbac.CreateSystemAPIToken(
			rbac.NewCreateSystemAPITokenParams().
				WithRole(o.RoleName).
				WithBody(&models.CreateAPITokenRequest{
					Name: o.Name,
				}),
			nil,
		); err != nil {
			return fmt.Errorf("create API token: %w", err)
		}
		payload = res.GetPayload()
	} else {
		var res *rbac.CreateProjectAPITokenCreated
		if res, err = apiClient.Rbac.CreateProjectAPIToken(
			rbac.NewCreateProjectAPITokenParams().
				WithProject(o.Project).
				WithRole(o.RoleName).
				WithBody(&models.CreateAPITokenRequest{
					Name: o.Name,
				}),
			nil,
		); err != nil {
			return fmt.Errorf("create API token: %w", err)
		}
		payload = res.GetPayload()
	}

	secretJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}
	var secret *corev1.Secret
	if err = json.Unmarshal(secretJSON, &secret); err != nil {
		return fmt.Errorf("unmarshal secret: %w", err)
	}

	// If user specified an output format (yaml, json, etc.), use it
	if o.OutputFlagSpecified != nil && o.OutputFlagSpecified() {
		printer, err := o.ToPrinter()
		if err != nil {
			return fmt.Errorf("new printer: %w", err)
		}
		return printer.PrintObj(secret, o.Out)
	}

	// Otherwise, print the token value clearly so users don't miss it
	tokenValue := string(secret.Data["token"])
	if tokenValue == "" {
		return fmt.Errorf("token data is empty")
	}

	fmt.Fprintf(o.Out, "Token created successfully!\n\n")
	fmt.Fprintf(o.Out, "IMPORTANT: Save this token securely. It will not be shown again.\n\n")
	fmt.Fprintf(o.Out, "Token: %s\n", tokenValue)

	return nil
}
