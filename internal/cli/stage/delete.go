package stage

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

func newDeleteCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete",
		Args:    cobra.ExactArgs(2),
		Example: "kargo stage delete (PROJECT) (NAME)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			project := strings.TrimSpace(args[0])
			if project == "" {
				return errors.New("project is required")
			}

			name := strings.TrimSpace(args[1])
			if name == "" {
				return errors.New("name is required")
			}

			client := svcv1alpha1connect.NewKargoServiceClient(http.DefaultClient, opt.ServerURL, opt.ClientOption)
			_, err := client.DeleteStage(ctx, connect.NewRequest(&v1alpha1.DeleteStageRequest{
				Project: project,
				Name:    name,
			}))
			if err != nil {
				return errors.Wrap(err, "delete stage")
			}
			_, _ = fmt.Fprintf(opt.IOStreams.Out, "Stage Deleted: %q", name)
			return nil
		},
	}
	return cmd
}
