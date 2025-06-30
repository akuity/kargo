package get

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/io"
	"github.com/akuity/kargo/internal/cli/kubernetes"
	"github.com/akuity/kargo/internal/cli/templates"
)

type getClusterConfigOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	*getOptions

	Config        config.CLIConfig
	ClientOptions client.Options
}

func newGetClusterConfigCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
	getOptions *getOptions,
) *cobra.Command {
	cmdOpts := &getClusterConfigOptions{
		Config:     cfg,
		IOStreams:  streams,
		getOptions: getOptions,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "clusterconfig [--no-headers]",
		Aliases: []string{"clusterconfigs"},
		Short:   "Display cluster configuration",
		Args:    cobra.NoArgs,
		Example: templates.Example(`
# Get cluster configuration
kargo get clusterconfig
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

// addFlags adds the flags for the get cluster config options to the provided
// command.
func (o *getClusterConfigOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.PrintFlags.AddFlags(cmd)
}

// run gets the cluster config from the server and prints it to the console.
func (o *getClusterConfigOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	res := make([]*kargoapi.ClusterConfig, 0, 1)
	resp, err := kargoSvcCli.GetClusterConfig(
		ctx,
		connect.NewRequest(
			&v1alpha1.GetClusterConfigRequest{},
		),
	)
	if err != nil {
		return fmt.Errorf("get cluster configuration: %w", err)
	}
	if resp.Msg.GetClusterConfig() != nil {
		res = append(res, resp.Msg.GetClusterConfig())
	}

	if err = printObjects(res, o.PrintFlags, o.IOStreams, o.NoHeaders); err != nil {
		return fmt.Errorf("print cluster configuration: %w", err)
	}
	return nil
}

func newClusterConfigTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		cfg := item.Object.(*kargoapi.ClusterConfig) // nolint: forcetypeassert

		rows[i] = metav1.TableRow{
			Cells: []any{
				cfg.Name,
				duration.HumanDuration(time.Since(cfg.CreationTimestamp.Time)),
			},
			Object: list.Items[i],
		}
	}
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Age", Type: "string"},
		},
		Rows: rows,
	}
}
