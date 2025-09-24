package config

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
)

type getProjectOptions struct {
	genericiooptions.IOStreams

	Config config.CLIConfig
}

func newGetProjectCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &getProjectOptions{
		Config:    cfg,
		IOStreams: streams,
	}

	cmd := &cobra.Command{
		Use:   "get-project",
		Short: "Display the default project",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Display the default project
kargo config get-project
`),
		RunE: func(*cobra.Command, []string) error {
			return cmdOpts.run()
		},
	}

	// Set the input/output streams for the command.
	io.SetIOStreams(cmd, cmdOpts.IOStreams)

	return cmd
}

// run prints the default project set in the CLI config.
func (o *getProjectOptions) run() error {
	if o.Config.Project == "" {
		return errors.New("default project is not set")
	}

	_, _ = fmt.Fprintf(o.Out, "%s\n", o.Config.Project)
	return nil
}
