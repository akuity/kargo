package delete

import (
	"context"
	goerrors "errors"
	"slices"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/kubernetes"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
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
		Example: `
# Delete a stage
kargo delete stage --project=my-project my-stage

# Delete multiple stages
kargo delete stage --project=my-project my-stage1 my-stage2

# Delete a stage in the default project
kargo config set-project my-project
kargo delete stage my-stage
`,
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
	cmd.SetIn(cmdOpts.IOStreams.In)
	cmd.SetOut(cmdOpts.IOStreams.Out)
	cmd.SetErr(cmdOpts.IOStreams.ErrOut)

	return cmd
}

// addFlags adds the flags for the delete stage options to the provided command.
func (o *deleteStageOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.PrintFlags.AddFlags(cmd)

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

	return goerrors.Join(errs...)
}

// run removes the stage(s) from the project based on the options.
func (o *deleteStageOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return errors.Wrap(err, "get client from config")
	}
	defer client.CloseIfPossible(kargoSvcCli)

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return errors.Wrap(err, "create printer")
	}

	var resErr error
	for _, name := range o.Names {
		if _, err := kargoSvcCli.DeleteStage(ctx, connect.NewRequest(&v1alpha1.DeleteStageRequest{
			Project: o.Project,
			Name:    name,
		})); err != nil {
			resErr = goerrors.Join(resErr, errors.Wrap(err, "Error"))
			continue
		}
		_ = printer.PrintObj(&kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: o.Project,
			},
		}, o.IOStreams.Out)
	}
	return resErr
}
