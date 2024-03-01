package delete

import (
	"context"
	goerrors "errors"
	"fmt"
	"slices"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type deleteProjectOptions struct {
	*option.Option
	Config config.CLIConfig

	Names []string
}

func newProjectCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmdOpts := &deleteProjectOptions{
		Option: opt,
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use:   "project (NAME ...)",
		Short: "Delete project by name",
		Args:  option.MinimumNArgs(1),
		Example: `
# Delete a project
kargo delete project my-project

# Delete multiple projects
kargo delete project my-project1 my-project2
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

	return cmd
}

// addFlags adds the flags for the delete project options to the provided
// command.
func (o *deleteProjectOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)
}

// complete sets the options from the command arguments.
func (o *deleteProjectOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *deleteProjectOptions) validate() error {
	if len(o.Names) == 0 {
		return errors.New("name is required")
	}
	return nil
}

// run removes the project(s) based on the options.
func (o *deleteProjectOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.Option)
	if err != nil {
		return errors.Wrap(err, "get client from config")
	}

	var resErr error
	for _, name := range o.Names {
		if _, err := kargoSvcCli.DeleteProject(ctx, connect.NewRequest(&v1alpha1.DeleteProjectRequest{
			Name: name,
		})); err != nil {
			resErr = goerrors.Join(resErr, errors.Wrap(err, "Error"))
			continue
		}
		_, _ = fmt.Fprintf(o.IOStreams.Out, "Project Deleted: %q\n", name)
	}
	return resErr
}
