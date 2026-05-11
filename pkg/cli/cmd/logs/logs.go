package logs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
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
	if o.Project == "" {
		return errors.New("project is required")
	}

	watchClient, err := client.GetWatchClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	logCh, errCh := watchClient.StreamAnalysisRunLogs(
		ctx,
		o.Project,
		o.Name,
		o.Metric,
		o.Container,
	)

	return o.displayLogs(ctx, logCh, errCh)
}

func (o *logsOptions) displayLogs(
	ctx context.Context,
	logCh <-chan string,
	errCh <-chan error,
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err, ok := <-errCh:
			if ok && err != nil {
				return fmt.Errorf("receive logs: %w", err)
			}
		case chunk, ok := <-logCh:
			if !ok {
				// Channel closed, stream complete
				return nil
			}
			_, _ = fmt.Fprint(o.Out, chunk)
		}
	}
}
