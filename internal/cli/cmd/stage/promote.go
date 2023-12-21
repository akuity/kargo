package stage

import (
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/utils/pointer"

	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type PromoteFlags struct {
	Freight string
}

func newPromoteCommand(opt *option.Option) *cobra.Command {
	var flag PromoteFlags
	cmd := &cobra.Command{
		Use:  "promote --project=project (STAGE) [(--freight=)freight-id]",
		Args: option.ExactArgs(1),
		Example: `
# Promote a freight to a stage for a specific project
kargo stage promote dev --project=my-project --freight=abc123

# Promote a freight to a stage for the default project
kargo config set project my-project
kargo stage promote dev --freight=abc123
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			project := opt.Project
			if project == "" {
				return errors.New("project is required")
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, opt)
			if err != nil {
				return err
			}

			stage := strings.TrimSpace(args[0])
			if stage == "" {
				return errors.New("name is required")
			}

			freight := strings.TrimSpace(flag.Freight)
			if freight == "" {
				// TODO: Get latest available freight if empty
				return errors.New("freight is required")
			}

			res, err := kargoSvcCli.PromoteStage(ctx, connect.NewRequest(&v1alpha1.PromoteStageRequest{
				Project: project,
				Name:    stage,
				Freight: freight,
			}))
			if err != nil {
				return errors.Wrap(err, "promote stage")
			}
			if pointer.StringDeref(opt.PrintFlags.OutputFormat, "") == "" {
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
		},
	}
	opt.PrintFlags.AddFlags(cmd)
	option.Freight(&flag.Freight)(cmd.Flags())
	option.Project(&opt.Project, opt.Project)(cmd.Flags())
	return cmd
}
