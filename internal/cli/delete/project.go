package delete

import (
	goerrors "errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func newProjectCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project [NAME]...",
		Short: "Delete project by name",
		Args:  cobra.MinimumNArgs(1),
		Example: `
# Delete project
kargo delete project my-project
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			kargoSvcCli, err := client.GetClientFromConfig(ctx, opt)
			if err != nil {
				return errors.New("get client from config")
			}

			var resErr error
			for _, name := range slices.Compact(args) {
				if _, err := kargoSvcCli.DeleteProject(ctx, connect.NewRequest(&v1alpha1.DeleteProjectRequest{
					Name: name,
				})); err != nil {
					resErr = goerrors.Join(resErr, errors.Wrap(err, "Error"))
					continue
				}
				_, _ = fmt.Fprintf(opt.IOStreams.Out, "Project Deleted: %q\n", name)
			}
			return resErr
		},
	}
	opt.PrintFlags.AddFlags(cmd)
	option.OptionalProject(opt.Project)(cmd.Flags())
	return cmd
}
