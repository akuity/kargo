package delete

import (
	goerrors "errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type deleteProjectOptions struct {
	*option.Option
}

// addFlags adds the flags for the delete project options to the provided
// command.
func (o *deleteProjectOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)
}

func newProjectCommand(
	cfg config.CLIConfig,
	opt *option.Option,
) *cobra.Command {
	cmdOpts := &deleteProjectOptions{Option: opt}

	cmd := &cobra.Command{
		Use:   "project [NAME]...",
		Short: "Delete project by name",
		Args:  option.MinimumNArgs(1),
		Example: `
# Delete project
kargo delete project my-project
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, cmdOpts.Option)
			if err != nil {
				return errors.Wrap(err, "get client from config")
			}

			var resErr error
			for _, name := range slices.Compact(args) {
				if _, err := kargoSvcCli.DeleteProject(ctx, connect.NewRequest(&v1alpha1.DeleteProjectRequest{
					Name: name,
				})); err != nil {
					resErr = goerrors.Join(resErr, errors.Wrap(err, "Error"))
					continue
				}
				_, _ = fmt.Fprintf(cmdOpts.IOStreams.Out, "Project Deleted: %q\n", name)
			}
			return resErr
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	return cmd
}
