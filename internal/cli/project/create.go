package project

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/bufbuild/connect-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

func newCreateCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create",
		Args:    cobra.ExactArgs(1),
		Example: "kargo project create (NAME)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			name := strings.TrimSpace(args[0])
			if name == "" {
				return errors.New("name is required")
			}

			client := svcv1alpha1connect.NewKargoServiceClient(http.DefaultClient, opt.ServerURL, opt.ClientOption)
			res, err := client.CreateProject(ctx, connect.NewRequest(&v1alpha1.CreateProjectRequest{
				Name: name,
			}))
			if err != nil {
				return errors.Wrap(err, "create project")
			}
			_, _ = fmt.Fprintf(opt.IOStreams.Out, "Project Created: %q", res.Msg.GetName())
			return nil
		},
	}
	return cmd
}
