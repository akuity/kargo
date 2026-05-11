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
	credclient "github.com/akuity/kargo/pkg/client/generated/credentials"
	"github.com/akuity/kargo/pkg/client/generated/models"
)

type createGenericCredentialsOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Project     string
	Shared      bool
	System      bool
	Name        string
	Description string
	SetValues   []string
	Data        map[string]string
}

func newGenericCredentialsCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &createGenericCredentialsOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("created").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use: `generic-credentials [--project=project | --shared | --system] NAME \
    [--description=description] \
    --set key=value [--set key=value ...]`,
		Aliases: []string{
			"generic-credential",
			"generic-creds",
			"generic-cred",
			"genericcredentials",
			"genericcredential",
			"genericcreds",
			"genericcred",
		},
		Short: "Create new generic credentials",
		Args:  cobra.ExactArgs(1),
		Example: templates.Example(`
# Create generic credentials in a project
kargo create generic-credentials --project=my-project my-credentials \
  --set API_KEY=my-api-key --set API_SECRET=my-secret

# Create generic credentials with a description
kargo create generic-credentials --project=my-project my-credentials \
  --description="API credentials for external service" \
  --set API_KEY=my-api-key

# Create shared generic credentials
kargo create generic-credentials --shared my-credentials \
  --set TOKEN=my-token

# Create system generic credentials
kargo create generic-credentials --system my-credentials \
  --set TOKEN=my-token

# Create generic credentials in the default project
kargo config set-project my-project
kargo create generic-credentials my-credentials \
  --set USERNAME=admin --set PASSWORD=secret
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

// addFlags adds the flags for the generic-credentials options to the provided
// command.
func (o *createGenericCredentialsOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project in which to create credentials. If not set, the default project will be used.",
	)
	option.Shared(
		cmd.Flags(), &o.Shared, false,
		"Whether to create shared credentials that can be used across all projects.",
	)
	option.System(
		cmd.Flags(), &o.System, false,
		"Whether to create system credentials.",
	)
	// project, shared, and system flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SharedFlag, option.SystemFlag)

	option.Description(cmd.Flags(), &o.Description, "Description of the credentials.")

	cmd.Flags().StringArrayVar(
		&o.SetValues,
		"set",
		nil,
		"Set a key-value pair in the credentials data (can be specified multiple times). Format: key=value",
	)

	if err := cmd.MarkFlagRequired("set"); err != nil {
		panic(fmt.Errorf("could not mark set flag as required: %w", err))
	}
}

// complete sets the options from the command arguments.
func (o *createGenericCredentialsOptions) complete(args []string) {
	o.Name = args[0]
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *createGenericCredentialsOptions) validate() error {
	var errs []error
	if o.Project == "" && !o.Shared && !o.System {
		errs = append(errs, fmt.Errorf(
			"one of %s, %s, or %s is required",
			option.ProjectFlag, option.SharedFlag, option.SystemFlag,
		))
	}

	// Parse and validate --set values
	o.Data = make(map[string]string)
	for _, kv := range o.SetValues {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			errs = append(errs, fmt.Errorf("invalid --set format %q: expected key=value", kv))
			continue
		}
		key, value := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if key == "" {
			errs = append(errs, fmt.Errorf("invalid --set format %q: key cannot be empty", kv))
			continue
		}
		o.Data[key] = value
	}

	if len(o.Data) == 0 {
		errs = append(errs, errors.New("at least one --set key=value is required"))
	}

	return errors.Join(errs...)
}

// run creates the generic credentials based on the options.
func (o *createGenericCredentialsOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	var resJSON []byte
	switch {
	case o.System:
		var res *credclient.CreateSystemGenericCredentialsCreated
		if res, err = apiClient.Credentials.CreateSystemGenericCredentials(
			credclient.NewCreateSystemGenericCredentialsParams().
				WithBody(&models.CreateGenericCredentialsRequest{
					Name:        o.Name,
					Description: o.Description,
					Data:        o.Data,
				}),
			nil,
		); err != nil {
			return fmt.Errorf("create system generic credentials: %w", err)
		}
		resJSON, err = json.Marshal(res.GetPayload())
	case o.Shared:
		var res *credclient.CreateSharedGenericCredentialsCreated
		if res, err = apiClient.Credentials.CreateSharedGenericCredentials(
			credclient.NewCreateSharedGenericCredentialsParams().
				WithBody(&models.CreateGenericCredentialsRequest{
					Name:        o.Name,
					Description: o.Description,
					Data:        o.Data,
				}),
			nil,
		); err != nil {
			return fmt.Errorf("create shared generic credentials: %w", err)
		}
		resJSON, err = json.Marshal(res.GetPayload())
	default:
		var res *credclient.CreateProjectGenericCredentialsCreated
		if res, err = apiClient.Credentials.CreateProjectGenericCredentials(
			credclient.NewCreateProjectGenericCredentialsParams().
				WithProject(o.Project).
				WithBody(&models.CreateGenericCredentialsRequest{
					Name:        o.Name,
					Description: o.Description,
					Data:        o.Data,
				}),
			nil,
		); err != nil {
			return fmt.Errorf("create project generic credentials: %w", err)
		}
		resJSON, err = json.Marshal(res.GetPayload())
	}
	// All three cases above end with marshaling the response payload, so we
	// can handle any of those potential errors here, in one place.
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}

	var secret *corev1.Secret
	if err = json.Unmarshal(resJSON, &secret); err != nil {
		return fmt.Errorf("unmarshal secret: %w", err)
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("new printer: %w", err)
	}
	return printer.PrintObj(secret, o.Out)
}
