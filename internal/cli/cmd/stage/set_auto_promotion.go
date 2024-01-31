package stage

import (
	"context"
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

func newEnableAutoPromotion(
	cfg config.CLIConfig,
	opt *option.Option,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "enable-auto-promotion --project=project (STAGE)",
		Args: option.ExactArgs(1),
		Example: `
# Enable auto-promotion on a stage for a specific project
kargo stage enable-auto-promotion --project=my-project dev

# Enable auto-promotion on a stage for the default project
kargo config set project my-project
kargo stage enable-auto-promotion dev
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			project := opt.Project
			if project == "" {
				return errors.New("project is required")
			}

			stage := strings.TrimSpace(args[0])
			if stage == "" {
				return errors.New("stage is required")
			}
			return setAutoPromotionForStage(ctx, cfg, opt, project, stage, true)
		},
	}
	opt.PrintFlags.AddFlags(cmd)
	option.Project(cmd.Flags(), opt, opt.Project)
	return cmd
}

func newDisableAutoPromotion(
	cfg config.CLIConfig,
	opt *option.Option,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:  "disable-auto-promotion --project=project (STAGE)",
		Args: option.ExactArgs(1),
		Example: `
# Disable auto-promotion on a stage for a specific project
kargo stage disable-auto-promotion --project=my-project dev

# Disable auto-promotion on a stage for the default project
kargo config set project my-project
kargo stage disable-auto-promotion dev
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			project := opt.Project
			if project == "" {
				return errors.New("project is required")
			}

			stage := strings.TrimSpace(args[0])
			if stage == "" {
				return errors.New("stage is required")
			}
			return setAutoPromotionForStage(ctx, cfg, opt, project, stage, false)
		},
	}
	opt.PrintFlags.AddFlags(cmd)
	option.Project(cmd.Flags(), opt, opt.Project)
	return cmd
}

func setAutoPromotionForStage(
	ctx context.Context,
	cfg config.CLIConfig,
	opt *option.Option,
	project string,
	stage string,
	enable bool,
) error {
	kargoClient, err := client.GetClientFromConfig(ctx, cfg, opt)
	if err != nil {
		return err
	}
	if _, err = kargoClient.SetAutoPromotionForStage(
		ctx,
		connect.NewRequest(&v1alpha1.SetAutoPromotionForStageRequest{
			Project: project,
			Stage:   stage,
			Enable:  enable,
		}),
	); err != nil {
		return errors.Wrapf(err, "set auto promotion for stage: %q", stage)
	}
	if ptr.Deref(opt.PrintFlags.OutputFormat, "") == "" {
		res := "Disabled"
		if enable {
			res = "Enabled"
		}
		fmt.Fprintf(opt.IOStreams.Out, "%s AutoPromotion for Stage %q\n", res, stage)
		return nil
	}
	return nil
}
