package delete

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/core"
)

type deleteProjectConfigOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
}

func newProjectConfigCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
) *cobra.Command {
	cmdOpts := &deleteProjectConfigOptions{
		Config:    cfg,
		IOStreams: streams,
		PrintFlags: genericclioptions.NewPrintFlags("deleted").
			WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "projectconfig [--project=project]",
		Aliases: []string{"projectconfigs"},
		Short:   "Delete project configuration",
		Args:    option.NoArgs,
		Example: templates.Example(`
# Delete project configuration for my-project
kargo delete projectconfig --project=my-project

# Delete project configuration for the default project
kargo config set-project my-project
kargo delete projectconfig
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmdOpts.run(cmd.Context())
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	// Set the input/output streams for the command.
	io.SetIOStreams(cmd, cmdOpts.IOStreams)

	return cmd
}

// addFlags adds the flags for the delete project config options to the provided
// command.
func (o *deleteProjectConfigOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(cmd.Flags(), &o.Project, o.Config.Project,
		"The Project for which to delete the configuration. If not set, the default project will be used.")
}

// run removes the project config from the project.
func (o *deleteProjectConfigOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("create printer: %w", err)
	}

	if _, err = apiClient.Core.DeleteProjectConfig(
		core.NewDeleteProjectConfigParams().
			WithProject(o.Project),
		nil,
	); err != nil {
		return fmt.Errorf("delete project configuration: %w", err)
	}

	if err = printer.PrintObj(
		&kargoapi.ProjectConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      o.Project,
				Namespace: o.Project,
			},
		}, o.Out); err != nil {
		return fmt.Errorf("print project configuration: %w", err)
	}

	return nil
}
