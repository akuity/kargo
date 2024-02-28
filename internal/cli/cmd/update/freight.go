package update

import (
	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func newUpdateFreightAliasCommand(
	cfg config.CLIConfig,
	opt *option.Option,
) *cobra.Command {
	var alias string
	cmd := &cobra.Command{
		Use:   "freight [--project=project] (NAME) --alias=alias",
		Args:  option.ExactArgs(1),
		Short: "Update (the alias of) a Freight",
		Example: `
# Update the alias of a freight for a specified project
kargo update freight --project=my-project abc123 --alias=my-new-alias

# Update the alias of a freight for the default project
kargo config set project my-project
kargo update freight abc123 --alias=my-new-alias
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			project := opt.Project
			if project == "" {
				return errors.New("project is required")
			}

			if alias == "" {
				return errors.New("alias is required")
			}

			kargoSvcCli, err := client.GetClientFromConfig(ctx, cfg, opt)
			if err != nil {
				return errors.Wrap(err, "get client from config")
			}

			if _, err = kargoSvcCli.UpdateFreightAlias(
				ctx,
				connect.NewRequest(
					&v1alpha1.UpdateFreightAliasRequest{
						Project: project,
						Freight: args[0],
						Alias:   alias,
					},
				),
			); err != nil {
				return errors.Wrap(err, "update freight alias")
			}

			return nil
		},
	}

	option.Project(cmd.Flags(), opt, opt.Project)

	cmd.Flags().StringVar(&alias, "alias", "", "A unique alias for the Freight")

	return cmd
}
