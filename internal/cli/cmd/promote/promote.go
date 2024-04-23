package promote

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/io"
	"github.com/akuity/kargo/internal/cli/kubernetes"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

type promotionOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Project       string
	FreightName   string
	FreightAlias  string
	Stage         string
	SubscribersOf string
	Wait          bool
}

func NewCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &promotionOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("promotion created").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use: "promote [--project=project] (--freight=freight | --freight-alias=alias) " +
			"(--stage=stage | --subscribers-of=stage)",
		Short: "Promote a piece of freight",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Promote a piece of freight specified by name to the QA stage
kargo promote --project=my-project --freight=abc123 --stage=qa

# Promote a piece of freight specified by alias to the QA stage
kargo promote --project=my-project --freight-alias=wonky-wombat --stage=qa

# Promote a piece of freight specified by name to subscribers of the QA stage
kargo promote --project=my-project --freight=abc123 --subscribers-of=qa

# Promote a piece of freight specified by alias to subscribers of the QA stage
kargo promote --project=my-project --freight-alias=wonky-wombat --subscribers-of=qa

# Promote a piece of freight specified by name to the QA stage in the default project
kargo config set-project my-project
kargo promote --freight=abc123 --stage=qa

# Promote a piece of freight specified by alias to the QA stage in the default project
kargo config set-project my-project
kargo promote --freight-alias=wonky-wombat --stage=qa

# Promote a piece of freight specified by name to subscribers of the QA stage in the default project
kargo config set-project my-project
kargo promote --freight=abc123 --subscribers-of=qa

# Promote a piece of freight specified by alias to subscribers of the QA stage in the default project
kargo config set-project my-project
kargo promote --freight-alias=wonky-wombat --subscribers-of=qas
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := cmdOpts.validate(); err != nil {
				return err
			}

			return cmdOpts.run(cmd.Context())
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	// Set the input/output streams for the command.
	io.SetIOStreams(cmd, cmdOpts.IOStreams)

	return cmd
}

// addFlags adds the flags for the promotion options to the provided command.
func (o *promotionOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.PrintFlags.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project the freight belongs to. If not set, the default project will be used.",
	)
	option.Freight(cmd.Flags(), &o.FreightName, "The name of piece of freight to promote.")
	option.FreightAlias(cmd.Flags(), &o.FreightAlias, "The alias of piece of freight to promote.")
	option.Stage(
		cmd.Flags(), &o.Stage,
		fmt.Sprintf(
			"The stage to promote the freight to. If set, --%s must not be set.",
			option.SubscribersOfFlag,
		),
	)
	option.SubscribersOf(
		cmd.Flags(), &o.SubscribersOf,
		fmt.Sprintf(
			"The stage whose subscribers freight should be promoted to. If set, --%s must not be set.",
			option.StageFlag,
		),
	)
	option.Wait(cmd.Flags(), &o.Wait, false, "Wait for the promotion(s) to complete.")

	cmd.MarkFlagsOneRequired(option.FreightFlag, option.FreightAliasFlag)
	cmd.MarkFlagsMutuallyExclusive(option.FreightFlag, option.FreightAliasFlag)

	cmd.MarkFlagsOneRequired(option.StageFlag, option.SubscribersOfFlag)
	cmd.MarkFlagsMutuallyExclusive(option.StageFlag, option.SubscribersOfFlag)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *promotionOptions) validate() error {
	var errs []error
	// While the flags are marked as required, a user could still provide an empty
	// string. This is a check to ensure that the flags are not empty.
	if o.Project == "" {
		errs = append(errs, fmt.Errorf("%s is required", option.ProjectFlag))
	}
	if o.FreightName == "" && o.FreightAlias == "" {
		errs = append(
			errs,
			fmt.Errorf("either %s or %s is required", option.FreightFlag, option.FreightAliasFlag),
		)
	}
	if o.Stage == "" && o.SubscribersOf == "" {
		errs = append(
			errs,
			fmt.Errorf("either %s or %s is required", option.StageFlag, option.SubscribersOfFlag),
		)
	}
	return errors.Join(errs...)
}

// run performs the promotion of the freight using the options.
func (o *promotionOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return fmt.Errorf("new printer: %w", err)
	}

	switch {
	case o.Stage != "":
		res, err := kargoSvcCli.PromoteToStage(
			ctx,
			connect.NewRequest(
				&v1alpha1.PromoteToStageRequest{
					Project:      o.Project,
					Freight:      o.FreightName,
					FreightAlias: o.FreightAlias,
					Stage:        o.Stage,
				},
			),
		)
		if err != nil {
			return fmt.Errorf("promote stage: %w", err)
		}
		if o.Wait {
			if err = waitForPromotion(ctx, kargoSvcCli, res.Msg.GetPromotion()); err != nil {
				return fmt.Errorf("wait for promotion: %w", err)
			}
		}
		_ = printer.PrintObj(res.Msg.GetPromotion(), o.IOStreams.Out)
		return nil
	case o.SubscribersOf != "":
		res, err := kargoSvcCli.PromoteToStageSubscribers(
			ctx,
			connect.NewRequest(
				&v1alpha1.PromoteToStageSubscribersRequest{
					Project:      o.Project,
					Freight:      o.FreightName,
					FreightAlias: o.FreightAlias,
					Stage:        o.SubscribersOf,
				},
			),
		)
		if err != nil {
			return fmt.Errorf("promote stage subscribers: %w", err)
		}
		if o.Wait {
			if err = waitForPromotions(ctx, kargoSvcCli, res.Msg.GetPromotions()...); err != nil {
				return fmt.Errorf("wait for promotions: %w", err)
			}
		}
		for _, p := range res.Msg.GetPromotions() {
			_ = printer.PrintObj(p, o.IOStreams.Out)
		}
		return nil
	}
	return nil
}

func waitForPromotions(
	ctx context.Context,
	kargoSvcCli svcv1alpha1connect.KargoServiceClient,
	p ...*kargoapi.Promotion,
) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, promo := range p {
		promo := promo
		g.Go(func() error {
			return waitForPromotion(ctx, kargoSvcCli, promo)
		})
	}
	return g.Wait()
}

func waitForPromotion(
	ctx context.Context,
	kargoSvcCli svcv1alpha1connect.KargoServiceClient,
	p *kargoapi.Promotion,
) error {
	if p == nil || p.Status.Phase.IsTerminal() {
		// No need to wait for a promotion that is already terminal.
		return nil
	}

	res, err := kargoSvcCli.WatchPromotion(ctx, connect.NewRequest(&v1alpha1.WatchPromotionRequest{
		Project: p.Namespace,
		Name:    p.Name,
	}))
	if err != nil {
		return fmt.Errorf("watch promotion: %w", err)
	}
	defer func() {
		if conn, connErr := res.Conn(); connErr == nil {
			_ = conn.CloseRequest()
		}
	}()
	for {
		if !res.Receive() {
			if err = res.Err(); err != nil {
				return fmt.Errorf("watch promotion: %w", err)
			}
			return errors.New("unexpected end of watch stream")
		}
		msg := res.Msg()
		if msg.GetPromotion().Status.Phase.IsTerminal() {
			return nil
		}
	}
}
