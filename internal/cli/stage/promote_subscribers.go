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

type PromoteSubscribersFlags struct {
	Freight string
}

func newPromoteSubscribersCommand(opt *option.Option) *cobra.Command {
	var flag PromoteSubscribersFlags
	cmd := &cobra.Command{
		Use:     "promote-subscribers",
		Args:    option.ExactArgs(2),
		Example: "kargo stage promote-subscribers (PROJECT) (NAME) [(--freight=)freight-id]",
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
				return errors.New("freight is required")
			}

			res, promoteErr := kargoSvcCli.PromoteSubscribers(ctx, connect.NewRequest(&v1alpha1.PromoteSubscribersRequest{
				Project: project,
				Stage:   name,
				Freight: freight,
			}))
			if pointer.StringDeref(opt.PrintFlags.OutputFormat, "") == "" {
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
		},
	}
	opt.PrintFlags.AddFlags(cmd)
	option.Freight(&flag.Freight)(cmd.Flags())
	return cmd
}
