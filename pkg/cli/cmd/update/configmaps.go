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
	"github.com/akuity/kargo/pkg/client/generated/core"
	"github.com/akuity/kargo/pkg/client/generated/models"
)

type updateConfigMapOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Shared      bool
	System      bool
	Project     string
	Name        string
	Description string
	SetValues   []string
	UnsetKeys   []string
	Data        map[string]string
	RemoveKeys  []string
}

func newUpdateConfigMapCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &updateConfigMapOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use: `configmap [--project=project | --shared | --system] NAME \
    [--description=description] \
    [--set key=value ...] [--unset key ...]`,
		Aliases: []string{"configmaps", "cm"},
		Short:   "Update a ConfigMap",
		Args:    cobra.ExactArgs(1),
		Example: templates.Example(`
# Update a key in a ConfigMap
kargo update configmap --project=my-project my-configmap \
  --set CONFIG_KEY=new-value

# Add a new key to a ConfigMap
kargo update configmap --project=my-project my-configmap \
  --set NEW_KEY=new-value

# Remove a key from a ConfigMap
kargo update configmap --project=my-project my-configmap \
  --unset OLD_KEY

# Update description of a ConfigMap
kargo update configmap --project=my-project my-configmap \
  --description="Updated description"

# Update multiple keys and remove others
kargo update configmap --project=my-project my-configmap \
  --set CONFIG_KEY=new-value --set ANOTHER_KEY=another-value --unset OLD_KEY

# Update a shared ConfigMap
kargo update configmap --shared my-configmap \
  --set CONFIG_KEY=new-value

# Update a system ConfigMap
kargo update configmap --system my-configmap \
  --set CONFIG_KEY=new-value

# Update a ConfigMap in the default project
kargo config set-project my-project
kargo update configmap my-configmap --set CONFIG_KEY=value
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

// addFlags adds the flags for the update configmap options to the provided command.
func (o *updateConfigMapOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project in which to update the ConfigMap. If not set, the default project will be used.",
	)
	option.Shared(
		cmd.Flags(), &o.Shared, false,
		"Whether to update a shared ConfigMap instead of a project-specific ConfigMap.",
	)
	option.System(
		cmd.Flags(), &o.System, false,
		"Whether to update a system ConfigMap.",
	)
	// project, shared, and system flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SharedFlag, option.SystemFlag)

	option.Description(cmd.Flags(), &o.Description, "Change the description of the ConfigMap.")

	cmd.Flags().StringArrayVar(
		&o.SetValues,
		"set",
		nil,
		"Set or update a key-value pair in the ConfigMap data (can be specified multiple times). Format: key=value",
	)

	cmd.Flags().StringArrayVar(
		&o.UnsetKeys,
		"unset",
		nil,
		"Remove a key from the ConfigMap data (can be specified multiple times)",
	)
}

// complete sets the options from the command arguments.
func (o *updateConfigMapOptions) complete(args []string) {
	o.Name = args[0]
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *updateConfigMapOptions) validate() error {
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

// run updates the ConfigMap based on the options.
func (o *updateConfigMapOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	// Build the request body - only include data if we have values to set
	var dataToSend map[string]string
	if len(o.Data) > 0 {
		dataToSend = o.Data
	}

	switch {
	case o.System:
		_, err = apiClient.Core.PatchSystemConfigMap(
			core.NewPatchSystemConfigMapParams().
				WithConfigmap(o.Name).
				WithBody(&models.PatchConfigMapRequest{
					Description: o.Description,
					Data:        dataToSend,
					RemoveKeys:  o.RemoveKeys,
				}),
			nil,
		)
		if err != nil {
			return fmt.Errorf("patch system ConfigMap: %w", err)
		}

		// Get the updated ConfigMap
		var res *core.GetSystemConfigMapOK
		if res, err = apiClient.Core.GetSystemConfigMap(
			core.NewGetSystemConfigMapParams().
				WithConfigmap(o.Name),
			nil,
		); err != nil {
			return fmt.Errorf("get system ConfigMap: %w", err)
		}

		return o.printConfigMap(res.GetPayload())
	case o.Shared:
		_, err = apiClient.Core.PatchSharedConfigMap(
			core.NewPatchSharedConfigMapParams().
				WithConfigmap(o.Name).
				WithBody(&models.PatchConfigMapRequest{
					Description: o.Description,
					Data:        dataToSend,
					RemoveKeys:  o.RemoveKeys,
				}),
			nil,
		)
		if err != nil {
			return fmt.Errorf("patch shared ConfigMap: %w", err)
		}

		// Get the updated ConfigMap
		var res *core.GetSharedConfigMapOK
		if res, err = apiClient.Core.GetSharedConfigMap(
			core.NewGetSharedConfigMapParams().
				WithConfigmap(o.Name),
			nil,
		); err != nil {
			return fmt.Errorf("get shared ConfigMap: %w", err)
		}

		return o.printConfigMap(res.GetPayload())
	default:
		_, err = apiClient.Core.PatchProjectConfigMap(
			core.NewPatchProjectConfigMapParams().
				WithProject(o.Project).
				WithConfigmap(o.Name).
				WithBody(&models.PatchConfigMapRequest{
					Description: o.Description,
					Data:        dataToSend,
					RemoveKeys:  o.RemoveKeys,
				}),
			nil,
		)
		if err != nil {
			return fmt.Errorf("patch project ConfigMap: %w", err)
		}

		// Get the updated ConfigMap
		res, err := apiClient.Core.GetProjectConfigMap(
			core.NewGetProjectConfigMapParams().
				WithProject(o.Project).
				WithConfigmap(o.Name),
			nil,
		)
		if err != nil {
			return fmt.Errorf("get project ConfigMap: %w", err)
		}

		return o.printConfigMap(res.GetPayload())
	}
}

func (o *updateConfigMapOptions) printConfigMap(payload any) error {
	configMapJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal ConfigMap: %w", err)
	}
	var configMap *corev1.ConfigMap
	if err = json.Unmarshal(configMapJSON, &configMap); err != nil {
		return fmt.Errorf("unmarshal ConfigMap: %w", err)
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("new printer: %w", err)
	}
	return printer.PrintObj(configMap, o.Out)
}
