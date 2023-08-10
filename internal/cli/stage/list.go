package stage

import (
	"strings"

	"github.com/bufbuild/connect-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/utils/pointer"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func newListCommand(opt *option.Option) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Args:    cobra.ExactArgs(1),
		Example: "kargo stage list (PROJECT)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			project := strings.TrimSpace(args[0])
			if project == "" {
				return errors.New("project is required")
			}

			client, err := client.GetClientFromConfig(opt)
			if err != nil {
				return err
			}

			res, err := client.ListStages(ctx, connect.NewRequest(&v1alpha1.ListStagesRequest{
				Project: project,
			}))
			if err != nil {
				return errors.Wrap(err, "list projects")
			}
			stages := &kubev1alpha1.StageList{
				Items: make([]kubev1alpha1.Stage, len(res.Msg.GetStages())),
			}
			for idx, stage := range res.Msg.GetStages() {
				stages.Items[idx] = *typesv1alpha1.FromStageProto(stage)
			}
			if pointer.StringDeref(opt.PrintFlags.OutputFormat, "") == "" {
				_ = printers.NewTablePrinter(printers.PrintOptions{}).PrintObj(stages, opt.IOStreams.Out)
				return nil
			}
			printer, err := opt.PrintFlags.ToPrinter()
			if err != nil {
				return errors.Wrap(err, "new printer")
			}
			return printer.PrintObj(stages, opt.IOStreams.Out)
		},
	}
	opt.PrintFlags.AddFlags(cmd)
	return cmd
}
