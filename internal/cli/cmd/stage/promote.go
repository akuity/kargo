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
		Use:     "promote",
		Args:    option.ExactArgs(2),
		Example: "kargo stage promote (PROJECT) (NAME) [(--freight=)freight-id]",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			kargoSvcCli, err := client.GetClientFromConfig(ctx, opt)
			if err != nil {
				return err
			}

			project := strings.TrimSpace(args[0])
			if project == "" {
				return errors.New("project is required")
			}
			name := strings.TrimSpace(args[1])
			if name == "" {
				return errors.New("name is required")
			}
			freight := strings.TrimSpace(flag.Freight)
			if freight == "" {
				// TODO: Get latest available freight if empty
				return errors.New("freight is required")
			}

			res, err := kargoSvcCli.PromoteStage(ctx, connect.NewRequest(&v1alpha1.PromoteStageRequest{
				Project: project,
				Name:    name,
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
	return cmd
}
