package approve

import (
	"context"
	goerrors "errors"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type approvalOptions struct {
	*option.Option
	Config config.CLIConfig

	FreightName  string
	FreightAlias string
	Stage        string
}

func NewCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmdOpts := &approvalOptions{
		Option: opt,
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use:   "approve [--project=project] (--freight=freight | --freight-alias=alias) --stage=stage",
		Short: "Manually approve a piece of freight for promotion to a stage",
		Args:  option.NoArgs,
		Example: `
# Approve a piece of freight specified by name for the QA stage
kargo approve --project=my-project --freight=abc1234 --stage=qa

# Approve a piece of freight specified by alias for the QA stage
kargo approve --project=my-project --freight-alias=wonky-wombat --stage=qa

# Approve a piece of freight specified by name for the QA stage in the default project
kargo config set-project my-project
kargo approve --freight=abc1234 --stage=qa

# Approve a piece of freight specified by alias for the QA stage in the default project
kargo config set-project my-project
kargo approve --freight-alias=wonky-wombat --stage=qa
`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := cmdOpts.validate(); err != nil {
				return err
			}

			return cmdOpts.run(cmd.Context())
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	return cmd
}

// addFlags adds the flags for the approval options to the provided command.
func (o *approvalOptions) addFlags(cmd *cobra.Command) {
	// TODO: Factor out server flags to a higher level (root?) as they are
	//   common to almost all commands.
	option.InsecureTLS(cmd.PersistentFlags(), o.Option)
	option.LocalServer(cmd.PersistentFlags(), o.Option)

	option.Project(
		cmd.Flags(), &o.Project, o.Project,
		"The project the freight belongs to. If not set, the default project will be used.",
	)
	option.Freight(cmd.Flags(), &o.FreightName, "The name of the freight to approve.")
	option.FreightAlias(cmd.Flags(), &o.FreightAlias, "The alias of the freight to approve.")
	option.Stage(cmd.Flags(), &o.Stage, "The stage for which to approve the freight.")

	if err := cmd.MarkFlagRequired(option.StageFlag); err != nil {
		panic(errors.Wrapf(err, "could not mark %s flag as required", option.StageFlag))
	}

	cmd.MarkFlagsOneRequired(option.FreightFlag, option.FreightAliasFlag)
	cmd.MarkFlagsMutuallyExclusive(option.FreightFlag, option.FreightAliasFlag)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *approvalOptions) validate() error {
	var errs []error
	// While the flags are marked as required, a user could still provide an empty
	// string. This is a check to ensure that the flags are not empty.
	if o.Project == "" {
		errs = append(errs, errors.Errorf("%s is required", option.ProjectFlag))
	}
	if o.FreightName == "" && o.FreightAlias == "" {
		errs = append(
			errs,
			errors.Errorf("either %s or %s is required", option.FreightFlag, option.FreightAliasFlag),
		)
	}
	if o.Stage == "" {
		errs = append(errs, errors.Errorf("%s is required", option.StageFlag))
	}
	return goerrors.Join(errs...)
}

// run performs the approval of a freight based on the options.
func (o *approvalOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.Option)
	if err != nil {
		return errors.Wrap(err, "get client from config")
	}

	if _, err = kargoSvcCli.ApproveFreight(
		ctx,
		connect.NewRequest(
			&v1alpha1.ApproveFreightRequest{
				Project: o.Project,
				Name:    o.FreightName,
				Alias:   o.FreightAlias,
				Stage:   o.Stage,
			},
		),
	); err != nil {
		return errors.Wrap(err, "approve freight")
	}
	return nil
}
