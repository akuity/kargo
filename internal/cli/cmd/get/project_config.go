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

type getProjectConfigOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	*getOptions

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
}

func newGetProjectConfigCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
	getOptions *getOptions,
) *cobra.Command {
	cmdOpts := &getProjectConfigOptions{
		Config:     cfg,
		IOStreams:  streams,
		getOptions: getOptions,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "projectconfiguration [PROJECT] [--no-headers]",
		Aliases: []string{"projectconfigurations", "projectconfig", "projectconfigs"},
		Short:   "Display project configuration",
		Args:    cobra.MaximumNArgs(1),
		Example: templates.Example(`
# Get project configuration for my-project
kargo get projectconfiguration my-project

# Get project configuration for the default project
kargo config set-project my-project
kargo get projectconfiguration
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdOpts.complete(args)
			return cmdOpts.run(cmd.Context())
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	// Set the input/output streams for the command.
	io.SetIOStreams(cmd, cmdOpts.IOStreams)

	return cmd
}

// addFlags adds the flags for the get project config options to the provided
// command.
func (o *getProjectConfigOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.PrintFlags.AddFlags(cmd)
}

// complete sets the options from the command arguments.
func (o *getProjectConfigOptions) complete(args []string) {
	o.Project = o.Config.Project
	if len(args) > 0 {
		o.Project = args[0]
	}
}

// run gets the project config from the server and prints it to the console.
func (o *getProjectConfigOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	res := make([]*kargoapi.ProjectConfig, 0, 1)
	resp, err := kargoSvcCli.GetProjectConfig(
		ctx,
		connect.NewRequest(
			&v1alpha1.GetProjectConfigRequest{
				Name: o.Project,
			},
		),
	)
	if err != nil {
		return fmt.Errorf("get project configuration: %w", err)
	}
	if resp.Msg.GetProjectConfig() != nil {
		res = append(res, resp.Msg.GetProjectConfig())
	}

	if err = printObjects(res, o.PrintFlags, o.IOStreams, o.NoHeaders); err != nil {
		return fmt.Errorf("print project configuration: %w", err)
	}
	return nil
}

func newProjectConfigTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		cfg := item.Object.(*kargoapi.ProjectConfig) // nolint: forcetypeassert

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
