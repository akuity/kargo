package create

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	sigyaml "sigs.k8s.io/yaml"

	kargosvcapi "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/io"
	"github.com/akuity/kargo/internal/cli/kubernetes"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
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
	o.PrintFlags.AddFlags(cmd)
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
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
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

	resp, err := kargoSvcCli.CreateResource(
		ctx,
		connect.NewRequest(
			&kargosvcapi.CreateResourceRequest{
				Manifest: projectBytes,
			},
		),
	)
	if err != nil {
		return fmt.Errorf("create resource: %w", err)
	}

	project = &kargoapi.Project{}
	projectBytes = resp.Msg.GetResults()[0].GetCreatedResourceManifest()
	if err = sigyaml.Unmarshal(projectBytes, project); err != nil {
		return fmt.Errorf("unmarshal project: %w", err)
	}

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return fmt.Errorf("new printer: %w", err)
	}
	return printer.PrintObj(project, o.IOStreams.Out)
}
