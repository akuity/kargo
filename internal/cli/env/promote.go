package env

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

type PromoteFlags struct {
	Project string
	State   string
}

func newPromoteCommand(opt *option.Option) *cobra.Command {
	var flag PromoteFlags
	cmd := &cobra.Command{
		Use:     "promote",
		Args:    cobra.ExactArgs(2),
		Example: "kargo environment promote (PROJECT) (NAME) [(--state=)state-id]",
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
			state := strings.TrimSpace(flag.State)
			if state == "" {
				// TODO: Get latest available state if empty
				return errors.New("state is required")
			}

			client := svcv1alpha1connect.NewKargoServiceClient(http.DefaultClient, opt.ServerURL, opt.ClientOption)
			res, err := client.PromoteEnvironment(ctx, connect.NewRequest(&v1alpha1.PromoteEnvironmentRequest{
				Project: project,
				Name:    name,
				State:   state,
			}))
			if err != nil {
				return errors.Wrap(err, "promote environment")
			}
			// TODO: Replace with console writer
			fmt.Printf("Promotion Created: %q", res.Msg.GetPromotion().GetMetadata().GetName())
			return nil
		},
	}
	option.State(&flag.State)(cmd.Flags())
	return cmd
}
