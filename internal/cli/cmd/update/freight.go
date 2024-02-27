package update

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

type updateFreightAliasOptions struct {
	*option.Option

	Alias string
}

// addFlags adds the flags for the update freight alias options to the provided
// command.
func (o *updateFreightAliasOptions) addFlags(cmd *cobra.Command) {
	option.Project(cmd.Flags(), &o.Project, o.Project,
		"The Project for which to list Promotions. If not set, the default project will be used.")
	cmd.Flags().StringVar(&o.Alias, "alias", "", "A unique alias for the Freight")

	if err := cmd.MarkFlagRequired("alias"); err != nil {
		panic(errors.Wrap(err, "could not mark alias flag as required"))
	}
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *updateFreightAliasOptions) validate() error {
	var errs []error

	if o.Project == "" {
		errs = append(errs, errors.New("project is required"))
	}

	// While the alias flag is marked as required, a user could still provide
	// an empty string. This is a check to ensure that the flag is not empty.
	if o.Alias == "" {
		errs = append(errs, errors.New("alias is required"))
	}

	return goerrors.Join(errs...)
}

func newUpdateFreightAliasCommand(
	cfg config.CLIConfig,
	opt *option.Option,
) *cobra.Command {
	cmdOpts := &updateFreightAliasOptions{Option: opt}

	cmd := &cobra.Command{
		Use:   "freight [--project=project] (NAME) --alias=alias",
		Args:  option.ExactArgs(1),
		Short: "Update (the alias of) a Freight",
		Example: `
# Update the alias of a freight for a specified project
kargo update freight --project=my-project abc123 --alias=my-new-alias

# Update the alias of a freight for the default project
kargo config set-project my-project
kargo update freight abc123 --alias=my-new-alias
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if err := cmdOpts.validate(); err != nil {
				return err
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, opt)
			if err != nil {
				return errors.Wrap(err, "get client from config")
			}

			if _, err = kargoSvcCli.UpdateFreightAlias(
				ctx,
				connect.NewRequest(
					&v1alpha1.UpdateFreightAliasRequest{
						Project: cmdOpts.Project,
						Freight: args[0],
						Alias:   cmdOpts.Alias,
					},
				),
			); err != nil {
				return errors.Wrap(err, "update freight alias")
			}

			return nil
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd)

	return cmd
}
