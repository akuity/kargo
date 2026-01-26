package update

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

type updateGenericCredentialsOptions struct {
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
	UnsetKeys   []string
	Data        map[string]string
	RemoveKeys  []string
}

func newUpdateGenericCredentialsCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &updateGenericCredentialsOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use: `generic-credentials [--project=project | --shared | --system] NAME \
    [--description=description] \
    [--set key=value ...] [--unset key ...]`,
		Aliases: []string{
			"generic-credential",
			"generic-creds",
			"generic-cred",
			"genericcredentials",
			"genericcredential",
			"genericcreds",
			"genericcred",
		},
		Short: "Update generic credentials",
		Args:  cobra.ExactArgs(1),
		Example: templates.Example(`
# Update a key in generic credentials
kargo update generic-credentials --project=my-project my-credentials \
  --set API_KEY=new-api-key

# Add a new key to generic credentials
kargo update generic-credentials --project=my-project my-credentials \
  --set NEW_KEY=new-value

# Remove a key from generic credentials
kargo update generic-credentials --project=my-project my-credentials \
  --unset OLD_KEY

# Update description of generic credentials
kargo update generic-credentials --project=my-project my-credentials \
  --description="Updated description"

# Update multiple keys and remove others
kargo update generic-credentials --project=my-project my-credentials \
  --set API_KEY=new-key --set API_SECRET=new-secret --unset OLD_TOKEN

# Update shared generic credentials
kargo update generic-credentials --shared my-credentials \
  --set TOKEN=new-token

# Update system generic credentials
kargo update generic-credentials --system my-credentials \
  --set TOKEN=new-token

# Update generic credentials in the default project
kargo config set-project my-project
kargo update generic-credentials my-credentials --set KEY=value
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

// addFlags adds the flags for the update generic-credentials options to the
// provided command.
func (o *updateGenericCredentialsOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project in which to update credentials. If not set, the default project will be used.",
	)
	option.Shared(
		cmd.Flags(), &o.Shared, false,
		"Whether to update shared credentials instead of project-specific credentials.",
	)
	option.System(
		cmd.Flags(), &o.System, false,
		"Whether to update system credentials instead of project-specific credentials.",
	)
	// project, shared, and system flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SharedFlag, option.SystemFlag)

	option.Description(cmd.Flags(), &o.Description, "Change the description of the credentials.")

	cmd.Flags().StringArrayVar(
		&o.SetValues,
		"set",
		nil,
		"Set or update a key-value pair in the credentials data (can be specified multiple times). Format: key=value",
	)

	cmd.Flags().StringArrayVar(
		&o.UnsetKeys,
		"unset",
		nil,
		"Remove a key from the credentials data (can be specified multiple times)",
	)
}

// complete sets the options from the command arguments.
func (o *updateGenericCredentialsOptions) complete(args []string) {
	o.Name = args[0]
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *updateGenericCredentialsOptions) validate() error {
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

	// Validate --unset keys
	o.RemoveKeys = make([]string, 0, len(o.UnsetKeys))
	for _, key := range o.UnsetKeys {
		if key == "" {
			errs = append(errs, errors.New("--unset key cannot be empty"))
			continue
		}
		o.RemoveKeys = append(o.RemoveKeys, key)
	}

	// At least one of --set, --unset, or --description must be provided
	if len(o.Data) == 0 && len(o.RemoveKeys) == 0 && o.Description == "" {
		errs = append(errs, errors.New("at least one of --set, --unset, or --description must be provided"))
	}

	return errors.Join(errs...)
}

// run updates the generic credentials based on the options.
func (o *updateGenericCredentialsOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	// Build the request body - only include data if we have values to set
	var dataToSend map[string]string
	if len(o.Data) > 0 {
		dataToSend = o.Data
	}

	var payload any

	switch {
	case o.System:
		_, err = apiClient.Credentials.PatchSystemGenericCredentials(
			credclient.NewPatchSystemGenericCredentialsParams().
				WithGenericCredentials(o.Name).
				WithBody(&models.PatchGenericCredentialsRequest{
					Description: o.Description,
					Data:        dataToSend,
					RemoveKeys:  o.RemoveKeys,
				}),
			nil,
		)
		if err != nil {
			return fmt.Errorf("patch system generic credentials: %w", err)
		}

		// Get the updated credentials
		var res *credclient.GetSystemGenericCredentialsOK
		if res, err = apiClient.Credentials.GetSystemGenericCredentials(
			credclient.NewGetSystemGenericCredentialsParams().
				WithGenericCredentials(o.Name),
			nil,
		); err != nil {
			return fmt.Errorf("get system generic credentials: %w", err)
		}
		payload = res.GetPayload()

	case o.Shared:
		_, err = apiClient.Credentials.PatchSharedGenericCredentials(
			credclient.NewPatchSharedGenericCredentialsParams().
				WithGenericCredentials(o.Name).
				WithBody(&models.PatchGenericCredentialsRequest{
					Description: o.Description,
					Data:        dataToSend,
					RemoveKeys:  o.RemoveKeys,
				}),
			nil,
		)
		if err != nil {
			return fmt.Errorf("patch shared generic credentials: %w", err)
		}

		// Get the updated credentials
		var res *credclient.GetSharedGenericCredentialsOK
		if res, err = apiClient.Credentials.GetSharedGenericCredentials(
			credclient.NewGetSharedGenericCredentialsParams().
				WithGenericCredentials(o.Name),
			nil,
		); err != nil {
			return fmt.Errorf("get shared generic credentials: %w", err)
		}
		payload = res.GetPayload()

	default:
		_, err = apiClient.Credentials.PatchProjectGenericCredentials(
			credclient.NewPatchProjectGenericCredentialsParams().
				WithProject(o.Project).
				WithGenericCredentials(o.Name).
				WithBody(&models.PatchGenericCredentialsRequest{
					Description: o.Description,
					Data:        dataToSend,
					RemoveKeys:  o.RemoveKeys,
				}),
			nil,
		)
		if err != nil {
			return fmt.Errorf("patch project generic credentials: %w", err)
		}

		// Get the updated credentials
		res, err := apiClient.Credentials.GetProjectGenericCredentials(
			credclient.NewGetProjectGenericCredentialsParams().
				WithProject(o.Project).
				WithGenericCredentials(o.Name),
			nil,
		)
		if err != nil {
			return fmt.Errorf("get project generic credentials: %w", err)
		}
		payload = res.GetPayload()
	}

	return o.printCredentials(payload)
}

func (o *updateGenericCredentialsOptions) printCredentials(payload any) error {
	credJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	var cred *corev1.Secret
	if err = json.Unmarshal(credJSON, &cred); err != nil {
		return fmt.Errorf("unmarshal credentials: %w", err)
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("new printer: %w", err)
	}
	return printer.PrintObj(cred, o.Out)
}
