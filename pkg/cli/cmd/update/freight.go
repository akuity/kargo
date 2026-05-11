package update

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/core"
)

type updateFreightAliasOptions struct {
	Config        config.CLIConfig
	ClientOptions client.Options

	Project  string
	Name     string
	OldAlias string
	NewAlias string
}

func newUpdateFreightAliasCommand(cfg config.CLIConfig) *cobra.Command {
	cmdOpts := &updateFreightAliasOptions{
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use:   "freight [--project=project] (--name=name | --old-alias=old-alias) --new-alias=new-alias",
		Short: "Update the alias of a piece of freight",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Update the alias of a piece of freight specified by name
kargo update freight --project=my-project --name=abc1234 --new-alias=frozen-fox

# Update the alias of a piece of freight specified by its existing alias
kargo update freight --project=my-project --old-alias=wonky-wombat --new-alias=frozen-fox

# Update the alias of a piece of freight specified by name in the default project
kargo config set-project my-project
kargo update freight --name=abc123 --new-alias=frozen-fox

# Update the alias of a piece of freight specified by its existing alias in the default project
kargo config set-project my-project
kargo update freight --old-alias=wonky-wombat --new-alias=frozen-fox
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

	return cmd
}

// addFlags adds the flags for the update freight alias options to the provided
// command.
func (o *updateFreightAliasOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project the freight belongs to. If not set, the default project will be used.",
	)
	option.Name(cmd.Flags(), &o.Name, "The name of the freight to be updated.")
	option.OldAlias(cmd.Flags(), &o.OldAlias, "The existing alias of the freight to be updated.")
	option.NewAlias(cmd.Flags(), &o.NewAlias, "The new alias to be assigned to the freight.")

	if err := cmd.MarkFlagRequired(option.NewAliasFlag); err != nil {
		panic(fmt.Errorf("could not mark %s flag as required: %w", option.NewAliasFlag, err))
	}

	cmd.MarkFlagsOneRequired(option.NameFlag, option.OldAliasFlag)
	cmd.MarkFlagsMutuallyExclusive(option.NameFlag, option.OldAliasFlag)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *updateFreightAliasOptions) validate() error {
	var errs []error
	// While the flags are marked as required, a user could still provide an empty
	// string. This is a check to ensure that the flags are not empty.
	if o.Project == "" {
		errs = append(errs, fmt.Errorf("%s is required", option.ProjectFlag))
	}
	if o.Name == "" && o.OldAlias == "" {
		errs = append(
			errs,
			fmt.Errorf("either %s or %s is required", option.NameFlag, option.OldAliasFlag),
		)
	}
	if o.NewAlias == "" {
		errs = append(errs, fmt.Errorf("%s is required", option.NewAliasFlag))
	}
	return errors.Join(errs...)
}

// run updates the freight alias using the options.
func (o *updateFreightAliasOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	// Use name if provided, otherwise use old alias
	freightNameOrAlias := o.Name
	if freightNameOrAlias == "" {
		freightNameOrAlias = o.OldAlias
	}

	if _, err = apiClient.Core.PatchFreightAlias(
		core.NewPatchFreightAliasParams().
			WithProject(o.Project).
			WithFreightNameOrAlias(freightNameOrAlias).
			WithNewAlias(o.NewAlias),
		nil,
	); err != nil {
		return fmt.Errorf("patch freight alias: %w", err)
	}
	return nil
}
