package logs

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/io"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
)

type logsOptions struct {
	genericiooptions.IOStreams

	Config        config.CLIConfig
	ClientOptions client.Options

	Project   string
	Name      string
	Metric    string
	Container string
}

func NewCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
) *cobra.Command {
	cmdOpts := &logsOptions{
		Config:    cfg,
		IOStreams: streams,
	}

	cmd := &cobra.Command{
		Use:   "logs [--project=project] NAME [--metric=metric] [--container=container]",
		Short: "View logs of completed AnalysisRuns (verifications) that utilize JobMetrics",
		Args:  cobra.ExactArgs(1),
		Example: templates.Example(`
# Show logs from an AnalysisRun with one JobMetric
kargo logs --project=my-project some-analysis-run

# Show logs from a specific JobMetric in an AnalysisRun with multiples
kargo logs --project=my-project some-analysis-run --metric=some-metric

# Show logs from a specific container in an AnalysisRun with one JobMetric
kargo logs --project=my-project some-analysis-run --container=some-container

# Show logs from a specific JobMetric and container
kargo logs --project=my-project some-analysis-run --metric=some-metric --container=some-container
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

// addFlags adds the flags for the logs options to the provided command.
func (o *logsOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project for which to get logs. If not set, the default project will be used.",
	)
	option.Metric(cmd.Flags(), &o.Metric, "A specific JobMetric of the AnalysisRun")
	option.Container(cmd.Flags(), &o.Container, "A specific container specified by the JobMetric")
}

// complete sets the options from the command arguments.
func (o *logsOptions) complete(args []string) {
	o.Name = strings.TrimSpace(args[0])
}

// run retrieves logs for the specified AnalysisRun.
func (o *logsOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	stream, err := kargoSvcCli.GetAnalysisRunLogs(
		ctx,
		connect.NewRequest(
			&v1alpha1.GetAnalysisRunLogsRequest{
				Namespace:     o.Project,
				Name:          o.Name,
				MetricName:    o.Metric,
				ContainerName: o.Container,
			},
		),
	)
	if err != nil {
		return fmt.Errorf("get logs from server: %w", err)
	}

	if err = o.displayLogs(ctx, stream); err != nil {
		return fmt.Errorf("display logs: %w", err)
	}

	return nil
}

func (o *logsOptions) displayLogs(
	ctx context.Context,
	stream *connect.ServerStreamForClient[v1alpha1.GetAnalysisRunLogsResponse],
) error {
	for stream.Receive() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		_, _ = fmt.Fprint(o.IOStreams.Out, stream.Msg().Chunk)
	}
	if err := stream.Err(); err != nil {
		return fmt.Errorf("receive logs: %w", err)
	}
	return nil
}
