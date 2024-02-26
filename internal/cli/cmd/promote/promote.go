package promote

import (
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/utils/ptr"

	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func NewCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	var freight string
	var stage string
	var subscribersOf string

	cmd := &cobra.Command{
		Use:   "promote [--project=project] --freight=freight-id [--stage=stage] [--subscribers-of=stage]",
		Short: "Manage the promotion of freight",
		Args:  option.NoArgs,
		Example: `
# Promote a freight to a stage for a specific project
kargo promote --project=my-project --freight=abc123 --stage=dev

# Promote a freight to subscribers of a stage for a specific project
kargo promote --project=my-project --freight=abc123 --subscribers-of=dev

# Promote a freight to a stage for the default project
kargo config set project my-project
kargo promote --freight=abc123 --stage=dev

# Promote a freight to subscribers of a stage for the default project
kargo config set project my-project
kargo promote --freight=abc123 --subscribers-of=dev
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			project := opt.Project
			if project == "" {
				return errors.New("project is required")
			}

			if freight == "" {
				return errors.New("freight is required")
			}

			if stage != "" && subscribersOf != "" {
				return errors.New("stage and subscribers-of can not be supplied simultaneously")
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, opt)
			if err != nil {
				return err
			}

			switch {
			case stage != "":
				res, err := kargoSvcCli.PromoteStage(ctx, connect.NewRequest(&v1alpha1.PromoteStageRequest{
					Project: project,
					Name:    stage,
					Freight: freight,
				}))
				if err != nil {
					return errors.Wrap(err, "promote stage")
				}
				if ptr.Deref(opt.PrintFlags.OutputFormat, "") == "" {
					fmt.Fprintf(opt.IOStreams.Out,
						"Promotion Created: %q\n", res.Msg.GetPromotion().GetMetadata().GetName())
					return nil
				}
				printer, err := opt.PrintFlags.ToPrinter()
				if err != nil {
					return errors.Wrap(err, "new printer")
				}
				promo := typesv1alpha1.FromPromotionProto(res.Msg.GetPromotion())
				_ = printer.PrintObj(promo, opt.IOStreams.Out)
				return nil
			case subscribersOf != "":
				res, promoteErr := kargoSvcCli.PromoteSubscribers(ctx, connect.NewRequest(&v1alpha1.PromoteSubscribersRequest{
					Project: project,
					Stage:   subscribersOf,
					Freight: freight,
				}))
				if ptr.Deref(opt.PrintFlags.OutputFormat, "") == "" {
					if res != nil && res.Msg != nil {
						for _, p := range res.Msg.GetPromotions() {
							fmt.Fprintf(opt.IOStreams.Out, "Promotion Created: %q\n", *p.Metadata.Name)
						}
					}
					if promoteErr != nil {
						return errors.Wrap(promoteErr, "promote subscribers")
					}
					return nil
				}

				printer, printerErr := opt.PrintFlags.ToPrinter()
				if printerErr != nil {
					return errors.Wrap(printerErr, "new printer")
				}
				for _, p := range res.Msg.GetPromotions() {
					kubeP := typesv1alpha1.FromPromotionProto(p)
					_ = printer.PrintObj(kubeP, opt.IOStreams.Out)
				}
				return promoteErr
			default:
				return errors.New("stage or subscribers-of is required")
			}
		},
	}

	opt.PrintFlags.AddFlags(cmd)
	option.InsecureTLS(cmd.PersistentFlags(), opt)
	option.LocalServer(cmd.PersistentFlags(), opt)

	option.Freight(cmd.Flags(), &freight)
	option.Stage(cmd.Flags(), &stage)
	option.SubscribersOf(cmd.Flags(), &subscribersOf)
	option.Project(cmd.Flags(), opt, opt.Project)

	return cmd
}
