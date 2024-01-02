package update

import (
	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func newUpdateFreightAliasCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "freight-alias --project=project NAME ALIAS",
		Args:    option.ExactArgs(2),
		Short:   "Update a freight alias",
		Example: "kargo update freight-alias --project=guestbook abc1234 foobar",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			project := opt.Project
			if project == "" {
				return errors.New("project is required")
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, opt)
			if err != nil {
				return errors.Wrap(err, "get client from config")
			}

			if _, err = kargoSvcCli.UpdateFreightAlias(
				ctx,
				connect.NewRequest(
					&v1alpha1.UpdateFreightAliasRequest{
						Project: project,
						Freight: args[0],
						Alias:   args[1],
					},
				),
			); err != nil {
				return errors.Wrap(err, "update freight alias")
			}

			return nil
		},
	}
	option.Project(&opt.Project, opt.Project)(cmd.Flags())
	return cmd
}
