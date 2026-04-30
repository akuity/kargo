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
	"github.com/akuity/kargo/pkg/client/generated/core"
	"github.com/akuity/kargo/pkg/client/generated/models"
)

type createConfigMapOptions struct {
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

func newConfigMapCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &createConfigMapOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("created").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use: `configmap [--project=project | --shared | --system] NAME \
    [--description=description] \
    --set key=value [--set key=value ...]`,
		Aliases: []string{"configmaps", "cm"},
		Short:   "Create a new ConfigMap",
		Args:    cobra.ExactArgs(1),
		Example: templates.Example(`
# Create a ConfigMap in a project
kargo create configmap --project=my-project my-configmap \
  --set CONFIG_KEY=config-value --set ANOTHER_KEY=another-value

# Create a ConfigMap with a description
kargo create configmap --project=my-project my-configmap \
  --description="Configuration for my application" \
  --set CONFIG_KEY=config-value

# Create a shared ConfigMap
kargo create configmap --shared my-configmap \
  --set CONFIG_KEY=config-value

# Create a system ConfigMap
kargo create configmap --system my-configmap \
  --set CONFIG_KEY=config-value

# Create a ConfigMap in the default project
kargo config set-project my-project
kargo create configmap my-configmap \
  --set CONFIG_KEY=config-value
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

// addFlags adds the flags for the configmap options to the provided command.
func (o *createConfigMapOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project in which to create the ConfigMap. If not set, the default project will be used.",
	)
	option.Shared(
		cmd.Flags(), &o.Shared, false,
		"Whether to create a shared ConfigMap that can be used across all projects.",
	)
	option.System(
		cmd.Flags(), &o.System, false,
		"Whether to create a system ConfigMap.",
	)
	// project, shared, and system flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SharedFlag, option.SystemFlag)

	option.Description(cmd.Flags(), &o.Description, "Description of the ConfigMap.")

	cmd.Flags().StringArrayVar(
		&o.SetValues,
		"set",
		nil,
		"Set a key-value pair in the ConfigMap data (can be specified multiple times). Format: key=value",
	)

	if err := cmd.MarkFlagRequired("set"); err != nil {
		panic(fmt.Errorf("could not mark set flag as required: %w", err))
	}
}

// complete sets the options from the command arguments.
func (o *createConfigMapOptions) complete(args []string) {
	o.Name = args[0]
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *createConfigMapOptions) validate() error {
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

// run creates the ConfigMap based on the options.
func (o *createConfigMapOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	var resJSON []byte
	switch {
	case o.System:
		var res *core.CreateSystemConfigMapCreated
		if res, err = apiClient.Core.CreateSystemConfigMap(
			core.NewCreateSystemConfigMapParams().
				WithBody(&models.CreateConfigMapRequest{
					Name:        o.Name,
					Description: o.Description,
					Data:        o.Data,
				}),
			nil,
		); err != nil {
			return fmt.Errorf("create system ConfigMap: %w", err)
		}
		resJSON, err = json.Marshal(res.GetPayload())
	case o.Shared:
		var res *core.CreateSharedConfigMapCreated
		if res, err = apiClient.Core.CreateSharedConfigMap(
			core.NewCreateSharedConfigMapParams().
				WithBody(&models.CreateConfigMapRequest{
					Name:        o.Name,
					Description: o.Description,
					Data:        o.Data,
				}),
			nil,
		); err != nil {
			return fmt.Errorf("create shared ConfigMap: %w", err)
		}
		resJSON, err = json.Marshal(res.GetPayload())
	default:
		var res *core.CreateProjectConfigMapCreated
		if res, err = apiClient.Core.CreateProjectConfigMap(
			core.NewCreateProjectConfigMapParams().
				WithProject(o.Project).
				WithBody(&models.CreateConfigMapRequest{
					Name:        o.Name,
					Description: o.Description,
					Data:        o.Data,
				}),
			nil,
		); err != nil {
			return fmt.Errorf("create project ConfigMap: %w", err)
		}
		resJSON, err = json.Marshal(res.GetPayload())
	}
	// All three cases above end with marshaling the response payload, so we
	// can handle any of those potential errors here, in one place.
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}

	var configMap *corev1.ConfigMap
	if err = json.Unmarshal(resJSON, &configMap); err != nil {
		return fmt.Errorf("unmarshal ConfigMap: %w", err)
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("new printer: %w", err)
	}
	return printer.PrintObj(configMap, o.Out)
}
