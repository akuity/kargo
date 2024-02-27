package approve

import (
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

	Freight string
	Stage   string
}

// addFlags adds the flags for the approval options to the provided command.
func (o *approvalOptions) addFlags(cmd *cobra.Command) {
	// TODO: Factor out server flags to a higher level (root?) as they are
	//   common to almost all commands.
	option.InsecureTLS(cmd.PersistentFlags(), o.Option)
	option.LocalServer(cmd.PersistentFlags(), o.Option)

	option.Project(cmd.Flags(), &o.Project, o.Project,
		"The Project the Freight belongs to. If not set, the default project will be used.")
	option.Freight(cmd.Flags(), &o.Freight, "The ID of the Freight to approve.")
	option.Stage(cmd.Flags(), &o.Stage, "The Stage to approve the Freight to.")

	if err := cmd.MarkFlagRequired(option.FreightFlag); err != nil {
		panic(errors.Wrap(err, "could not mark freight flag as required"))
	}
	if err := cmd.MarkFlagRequired(option.StageFlag); err != nil {
		panic(errors.Wrap(err, "could not mark stage flag as required"))
	}
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *approvalOptions) validate() error {
	var errs []error

	if o.Project == "" {
		errs = append(errs, errors.New("project is required"))
	}

	// While the freight and stage flags are marked as required, a user could
	// still provide an empty string. This is a check to ensure that the flags
	// are not empty.
	if o.Freight == "" {
		errs = append(errs, errors.New("freight is required"))
	}
	if o.Stage == "" {
		errs = append(errs, errors.New("stage is required"))
	}

	return goerrors.Join(errs...)
}

func NewCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmdOpts := &approvalOptions{Option: opt}

	cmd := &cobra.Command{
		Use:     "approve --project=project --freight=freight --stage=stage",
		Short:   "Manually approve freight for promotion to a stage",
		Example: "kargo approve --project=project --freight=abc1234 --stage=qa",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			if err := cmdOpts.validate(); err != nil {
				return err
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, cmdOpts.Option)
			if err != nil {
				return errors.Wrap(err, "get client from config")
			}

			if _, err = kargoSvcCli.ApproveFreight(
				ctx,
				connect.NewRequest(
					&v1alpha1.ApproveFreightRequest{
						Project: cmdOpts.Project,
						Id:      cmdOpts.Freight,
						Stage:   cmdOpts.Stage,
					},
				),
			); err != nil {
				return errors.Wrap(err, "approve freight")
			}
			return nil
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	return cmd
}
