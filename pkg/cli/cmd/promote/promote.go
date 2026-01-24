package promote

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/core"
	"github.com/akuity/kargo/pkg/client/generated/models"
	"github.com/akuity/kargo/pkg/client/watch"
)

type promotionOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Project        string
	FreightName    string
	FreightAlias   string
	Promotion      string
	Stage          string
	DownstreamFrom string
	Abort          bool
	Wait           bool
}

func NewCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &promotionOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("promotion created").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use: "promote [--project=project] (--freight=freight | --freight-alias=alias | --name=name) " +
			"[(--stage=stage | --downstream-from=stage) | --abort]",
		Short: "Promote a piece of freight",
		Args:  option.NoArgs,
		// nolint: lll
		Example: templates.Example(`
# Promote a piece of freight specified by name to the QA stage
kargo promote --project=my-project --freight=abc123 --stage=qa

# Promote a piece of freight specified by alias to the QA stage
kargo promote --project=my-project --freight-alias=wonky-wombat --stage=qa

# Promote a piece of freight specified by name to stages immediately downstream from the QA stage
kargo promote --project=my-project --freight=abc123 --downstream-from=qa

# Promote a piece of freight specified by alias to stages immediately downstream from the QA stage
kargo promote --project=my-project --freight-alias=wonky-wombat --downstream-from=qa

# Abort a Promotion by name
kargo promote --project=my-project --name=my-promotion --abort

# Promote a piece of freight specified by name to the QA stage in the default project
kargo config set-project my-project
kargo promote --freight=abc123 --stage=qa

# Promote a piece of freight specified by alias to the QA stage in the default project
kargo config set-project my-project
kargo promote --freight-alias=wonky-wombat --stage=qa

# Promote a piece of freight specified by name to stages immediately downstream from the QA stage in the default project
kargo config set-project my-project
kargo promote --freight=abc123 --downstream-from=qa

# Promote a piece of freight specified by alias to stages immediately downstream from of the QA stage in the default project
kargo config set-project my-project
kargo promote --freight-alias=wonky-wombat --downstream-from=qas
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
	o.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project the freight belongs to. If not set, the default project will be used.",
	)
	option.Freight(cmd.Flags(), &o.FreightName, "The name of piece of freight to promote.")
	option.FreightAlias(cmd.Flags(), &o.FreightAlias, "The alias of piece of freight to promote.")
	option.Name(cmd.Flags(), &o.Promotion, "The name of a promotion. Only used when aborting a promotion.")
	option.Stage(
		cmd.Flags(), &o.Stage,
		fmt.Sprintf(
			"The stage to promote the freight to. If set, --%s must not be set.",
			option.DownstreamFromFlag,
		),
	)
	option.DownstreamFrom(
		cmd.Flags(), &o.DownstreamFrom,
		fmt.Sprintf(
			"The stage whose immediately downstream stages freight should be promoted to. If set, --%s must not be set.",
			option.StageFlag,
		),
	)
	option.Abort(cmd.Flags(), &o.Abort, false, fmt.Sprintf(
		"Abort a non-terminal promotion. If set, --%s must be set.", option.NameFlag,
	))
	option.Wait(cmd.Flags(), &o.Wait, false, "Wait for the promotion(s) to complete.")

	cmd.MarkFlagsOneRequired(option.FreightFlag, option.FreightAliasFlag, option.NameFlag)
	cmd.MarkFlagsMutuallyExclusive(option.FreightFlag, option.FreightAliasFlag, option.NameFlag)

	cmd.MarkFlagsOneRequired(option.StageFlag, option.DownstreamFromFlag, option.AbortFlag)
	cmd.MarkFlagsMutuallyExclusive(option.StageFlag, option.DownstreamFromFlag, option.AbortFlag)

	cmd.MarkFlagsRequiredTogether(option.NameFlag, option.AbortFlag)
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
	if o.Abort {
		if o.Promotion == "" {
			errs = append(errs, fmt.Errorf("%s is required when aborting a promotion", option.NameFlag))
		}
	} else {
		if o.FreightName == "" && o.FreightAlias == "" {
			errs = append(
				errs,
				fmt.Errorf("either %s or %s is required", option.FreightFlag, option.FreightAliasFlag),
			)
		}
		if o.Stage == "" && o.DownstreamFrom == "" {
			errs = append(
				errs,
				fmt.Errorf("either %s or %s is required", option.StageFlag, option.DownstreamFromFlag),
			)
		}
	}
	return errors.Join(errs...)
}

// run performs the promotion of the freight using the options.
func (o *promotionOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("new printer: %w", err)
	}

	switch {
	case o.Abort:
		if _, err = apiClient.Core.AbortPromotion(
			core.NewAbortPromotionParams().WithProject(o.Project).WithPromotion(o.Promotion),
			nil,
		); err != nil {
			return fmt.Errorf("abort promotion: %w", err)
		}
		return nil
	case o.Stage != "":
		var res *core.PromoteToStageCreated
		if res, err = apiClient.Core.PromoteToStage(
			core.NewPromoteToStageParams().
				WithProject(o.Project).
				WithStage(o.Stage).
				WithBody(&models.PromoteToStageRequest{
					Freight:      o.FreightName,
					FreightAlias: o.FreightAlias,
				}),
			nil,
		); err != nil {
			return err
		}
		promoJSON, err := json.Marshal(res.Payload)
		if err != nil {
			return fmt.Errorf("marshal promotion: %w", err)
		}
		promo := &kargoapi.Promotion{}
		if err = json.Unmarshal(promoJSON, promo); err != nil {
			return fmt.Errorf("unmarshal promotion: %w", err)
		}
		if o.Wait {
			if err = o.waitForPromotion(ctx, nil, promo); err != nil {
				return fmt.Errorf("wait for promotion: %w", err)
			}
		}
		_ = printer.PrintObj(promo, o.Out)
		return nil
	case o.DownstreamFrom != "":
		res, err := apiClient.Core.PromoteDownstream(
			core.NewPromoteDownstreamParams().
				WithProject(o.Project).
				WithStage(o.DownstreamFrom).
				WithBody(&models.PromoteDownstreamRequest{
					Freight:      o.FreightName,
					FreightAlias: o.FreightAlias,
				}),
			nil,
		)
		if err != nil {
			return err
		}
		var promotions []*kargoapi.Promotion
		promotionsJSON, err := json.Marshal(res.Payload)
		if err != nil {
			return fmt.Errorf("marshal promotions: %w", err)
		}
		if err = json.Unmarshal(promotionsJSON, &promotions); err != nil {
			return fmt.Errorf("unmarshal promotions: %w", err)
		}
		if o.Wait {
			if err = o.waitForPromotions(ctx, promotions...); err != nil {
				return fmt.Errorf("wait for promotion: %w", err)
			}
		}
		for _, p := range promotions {
			_ = printer.PrintObj(p, o.Out)
		}
		return nil
	}
	return nil
}

func (o *promotionOptions) waitForPromotions(
	ctx context.Context,
	p ...*kargoapi.Promotion,
) error {
	watchClient, err := client.GetWatchClientFromConfig(
		ctx,
		o.Config,
		o.ClientOptions,
	)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}
	g, ctx := errgroup.WithContext(ctx)
	for _, promo := range p {
		g.Go(func() error {
			return o.waitForPromotion(ctx, watchClient, promo)
		})
	}
	return g.Wait()
}

func (o *promotionOptions) waitForPromotion(
	ctx context.Context,
	watchClient *watch.Client,
	p *kargoapi.Promotion,
) error {
	if p == nil || p.Status.Phase.IsTerminal() {
		// No need to wait for a promotion that is already terminal.
		return nil
	}

	if watchClient == nil {
		var err error
		if watchClient, err = client.GetWatchClientFromConfig(
			ctx,
			o.Config,
			o.ClientOptions,
		); err != nil {
			return fmt.Errorf("get client from config: %w", err)
		}
	}

	eventCh, errCh := watchClient.WatchPromotion(ctx, p.Namespace, p.Name)
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				select {
				case err := <-errCh:
					if err != nil {
						return fmt.Errorf("watch promotion: %w", err)
					}
				default:
				}
				return errors.New("unexpected end of watch stream")
			}
			if event.Object != nil && event.Object.Status.Phase.IsTerminal() {
				return nil
			}
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("watch promotion: %w", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
