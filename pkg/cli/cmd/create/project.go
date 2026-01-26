package create

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	sigyaml "sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/resources"
)

type createProjectOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Name string
}

func newProjectCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &createProjectOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("created").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:   "project NAME",
		Short: "Create a project",
		Args:  option.ExactArgs(1),
		Example: templates.Example(`
# Create project
kargo create project my-project
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

// addFlags adds the flags for the create project options to the provided command.
func (o *createProjectOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)
}

// complete sets the options from the command arguments.
func (o *createProjectOptions) complete(args []string) {
	o.Name = strings.TrimSpace(args[0])
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *createProjectOptions) validate() error {
	if o.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// run creates a project using the provided options.
func (o *createProjectOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	project := &kargoapi.Project{
		TypeMeta: metav1.TypeMeta{
			APIVersion: kargoapi.GroupVersion.String(),
			Kind:       "Project",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: o.Name,
		},
	}
	projectBytes, err := sigyaml.Marshal(project)
	if err != nil {
		return fmt.Errorf("marshal project: %w", err)
	}

	res, err := apiClient.Resources.CreateResource(
		resources.NewCreateResourceParams().
			WithManifest(string(projectBytes)),
		nil,
	)
	if err != nil {
		return fmt.Errorf("create resource: %w", err)
	}

	if len(res.Payload.Results) == 0 || res.Payload.Results[0].Error != "" {
		if len(res.Payload.Results) > 0 {
			return errors.New(res.Payload.Results[0].Error)
		}
		return errors.New("no results returned")
	}

	// Convert map to JSON then unmarshal to Project
	manifestJSON, err := json.Marshal(res.Payload.Results[0].CreatedResourceManifest)
	if err != nil {
		return fmt.Errorf("marshal created manifest: %w", err)
	}

	project = &kargoapi.Project{}
	if err = json.Unmarshal(manifestJSON, project); err != nil {
		return fmt.Errorf("unmarshal project: %w", err)
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("new printer: %w", err)
	}
	return printer.PrintObj(project, o.Out)
}
