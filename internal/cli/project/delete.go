package project

import (
	"fmt"
	"strings"

	"github.com/bufbuild/connect-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func newDeleteCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete",
		Args:    cobra.ExactArgs(1),
		Example: "kargo project delete (NAME)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			name := strings.TrimSpace(args[0])
			if name == "" {
				return errors.New("name is required")
			}

			client, err := client.GetClientFromConfig(ctx, opt)
			if err != nil {
				return err
			}

			if _, err := client.DeleteProject(ctx, connect.NewRequest(&v1alpha1.DeleteProjectRequest{
				Name: name,
			})); err != nil {
				return errors.Wrap(err, "delete project")
			}
			_, _ = fmt.Fprintf(opt.IOStreams.Out, "Project Deleted: %q", name)
			return nil
		},
	}
	return cmd
}
