package stage

import (
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/utils/ptr"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func newPromoteCommand(
	cfg config.CLIConfig,
	opt *option.Option,
) *cobra.Command {
	var freight string
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

			kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, opt)
			if err != nil {
				return err
			}

			stage := strings.TrimSpace(args[0])
			if stage == "" {
				return errors.New("name is required")
			}

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
			if ptr.Deref(opt.PrintFlags.OutputFormat, "") == "" {
				fmt.Fprintf(opt.IOStreams.Out,
					"Promotion Created: %q\n", res.Msg.GetPromotion().Name)
				return nil
			}
			printer, err := opt.PrintFlags.ToPrinter()
			if err != nil {
				return errors.Wrap(err, "new printer")
			}
			_ = printer.PrintObj(res.Msg.GetPromotion(), opt.IOStreams.Out)
			return nil
		},
	}
	opt.PrintFlags.AddFlags(cmd)
	option.Freight(cmd.Flags(), &freight)
	option.Project(cmd.Flags(), opt, opt.Project)
	return cmd
}
