package delete

import (
	"errors"
	"fmt"
	"strings"

	pkgerrors "github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/option"
)

func NewCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [--project=project] (RESOURCE) [NAME]...",
		Short: "Delete resources by resources and names",
		Args:  cobra.MinimumNArgs(2),
		Example: `
# Delete stage
kargo delete stage --project=my-project my-stage

# Delete project
kargo delete project my-project
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			kargoSvcCli, err := client.GetClientFromConfig(ctx, opt)
			if err != nil {
				return pkgerrors.New("get client from config")
			}

			resource := strings.ToLower(strings.TrimSpace(args[0]))
			if resource == "" {
				return pkgerrors.New("resource is required")
			}

			var resErr error
			switch resource {
			case "project", "projects":
				for _, name := range slices.Compact(args[1:]) {
					if err := deleteProject(ctx, kargoSvcCli, name); err != nil {
						resErr = errors.Join(resErr, pkgerrors.Wrap(err, "Error"))
						continue
					}
					_, _ = fmt.Fprintf(opt.IOStreams.Out, "Project Deleted: %q\n", name)
				}
			case "stage", "stages":
				project := opt.Project.OrElse("")
				if project == "" {
					return errors.New("project is required")
				}
				for _, name := range slices.Compact(args[1:]) {
					if err := deleteStage(ctx, kargoSvcCli, project, name); err != nil {
						resErr = errors.Join(resErr, pkgerrors.Wrap(err, "Error"))
						continue
					}
					_, _ = fmt.Fprintf(opt.IOStreams.Out, "Stage Deleted: %q\n", name)
				}
			default:
				return pkgerrors.Errorf("unknown resource %q", resource)
			}
			return resErr
		},
	}
	option.OptionalProject(opt.Project)(cmd.Flags())
	return cmd
}
