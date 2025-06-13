package delete

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/io"
	"github.com/akuity/kargo/internal/cli/kubernetes"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
)

type deleteClusterConfigOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
}

func newClusterConfigCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
) *cobra.Command {
	cmdOpts := &deleteClusterConfigOptions{
		Config:    cfg,
		IOStreams: streams,
		PrintFlags: genericclioptions.NewPrintFlags("deleted").
			WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "clusterconfig",
		Aliases: []string{"clusterconfigs"},
		Short:   "Delete cluster configuration",
		Args:    option.NoArgs,
		Example: templates.Example(`
# Delete the cluster configuration
kargo delete clusterconfig
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
func (o *deleteClusterConfigOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.PrintFlags.AddFlags(cmd)
}

// run removes the project config from the project.
func (o *deleteClusterConfigOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return fmt.Errorf("create printer: %w", err)
	}

	if _, err = kargoSvcCli.DeleteClusterConfig(
		ctx,
		connect.NewRequest(&v1alpha1.DeleteClusterConfigRequest{}),
	); err != nil {
		return fmt.Errorf("delete cluster configuration: %w", err)
	}

	if err = printer.PrintObj(
		&kargoapi.ClusterConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: o.Project,
			},
		}, o.IOStreams.Out); err != nil {
		return fmt.Errorf("print project configuration: %w", err)
	}

	return nil
}
