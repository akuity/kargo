package delete

import (
	"context"
	"errors"
	"fmt"
	"slices"

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

type deleteStageOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
	Names   []string
}

func newStageCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &deleteStageOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("deleted").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:   "stage [--project=project] (NAME ...)",
		Short: "Delete stage by name",
		Args:  option.MinimumNArgs(1),
		Example: templates.Example(`
# Delete a stage
kargo delete stage --project=my-project my-stage

# Delete multiple stages
kargo delete stage --project=my-project my-stage1 my-stage2

# Delete a stage in the default project
kargo config set-project my-project
kargo delete stage my-stage
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

// addFlags adds the flags for the delete stage options to the provided command.
func (o *deleteStageOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(cmd.Flags(), &o.Project, o.Config.Project,
		"The Project for which to delete Stages. If not set, the default project will be used.")
}

// complete sets the options from the command arguments.
func (o *deleteStageOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *deleteStageOptions) validate() error {
	var errs []error

	if o.Project == "" {
		errs = append(errs, errors.New("project is required"))
	}

	if len(o.Names) == 0 {
		errs = append(errs, errors.New("name is required"))
	}

	return errors.Join(errs...)
}

// run removes the stage(s) from the project based on the options.
func (o *deleteStageOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("create printer: %w", err)
	}

	var errs []error
	for _, name := range o.Names {
		if _, err := apiClient.Core.DeleteStage(
			core.NewDeleteStageParams().
				WithProject(o.Project).
				WithStage(name),
			nil,
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
