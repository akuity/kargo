package approve

import (
	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func NewCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	var freight, stage string
	cmd := &cobra.Command{
		Use:     "approve --project=project --freight=freight --stage=stage",
		Short:   "Manually approve freight for promotion to a stage",
		Example: "kargo approve --project=project --freight=abc1234 --stage=qa",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			project := opt.Project
			if project == "" {
				return errors.New("project is required")
			}
			if freight == "" {
				return errors.New("freight is required")
			}
			if stage == "" {
				return errors.New("stage is required")
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, opt)
			if err != nil {
				return errors.Wrap(err, "get client from config")
			}

			if _, err = kargoSvcCli.ApproveFreight(
				ctx,
				connect.NewRequest(
					&v1alpha1.ApproveFreightRequest{
						Project: project,
						Id:      freight,
						Stage:   stage,
					},
				),
			); err != nil {
				return errors.Wrap(err, "approve freight")
			}
			return nil
		},
	}
	option.Project(cmd.Flags(), opt, opt.Project)
	option.Freight(cmd.Flags(), &freight)
	option.Stage(cmd.Flags(), &stage)
	return cmd
}
