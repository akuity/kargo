package unreject

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/core"
)

type freightOptions struct {
	genericiooptions.IOStreams

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
	Name    string
	Alias   string
}

func newFreightCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &freightOptions{
		Config:    cfg,
		IOStreams: streams,
	}

	cmd := &cobra.Command{
		Use:   "freight [--project=project] (--name=name | --alias=alias)",
		Short: "Clear rejected status from freight by name or alias",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Clear rejection from a piece of freight by name
kargo unreject freight --project=my-project --name=abc123

# Clear rejection from a piece of freight by alias
kargo unreject freight --project=my-project --alias=wonky-name

# Clear rejection from a piece of freight in the default project
kargo config set-project my-project
kargo unreject freight --name=abc123
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := cmdOpts.validate(); err != nil {
				return err
			}
			return cmdOpts.run(cmd.Context())
		},
	}

	cmdOpts.addFlags(cmd)
	io.SetIOStreams(cmd, cmdOpts.IOStreams)
	return cmd
}

func (o *freightOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	option.Project(
		cmd.Flags(),
		&o.Project,
		o.Config.Project,
		"The project the freight belongs to. If not set, the default project will be used.",
	)
	option.Name(cmd.Flags(), &o.Name, "The name of the freight to unreject.")
	option.Alias(cmd.Flags(), &o.Alias, "The alias of the freight to unreject.")
	cmd.MarkFlagsOneRequired(option.NameFlag, option.AliasFlag)
	cmd.MarkFlagsMutuallyExclusive(option.NameFlag, option.AliasFlag)
}

func (o *freightOptions) validate() error {
	var errs []error
	o.Project = strings.TrimSpace(o.Project)
	o.Name = strings.TrimSpace(o.Name)
	o.Alias = strings.TrimSpace(o.Alias)

	if o.Project == "" {
		errs = append(errs, fmt.Errorf("%s is required", option.ProjectFlag))
	}
	if o.Name == "" && o.Alias == "" {
		errs = append(
			errs,
			fmt.Errorf("either %s or %s is required", option.NameFlag, option.AliasFlag),
		)
	}
	return errors.Join(errs...)
}

func (o *freightOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	nameOrAlias := o.Name
	if nameOrAlias == "" {
		nameOrAlias = o.Alias
	}

	if _, err = apiClient.Core.ClearFreightRejection(
		core.NewClearFreightRejectionParams().
			WithProject(o.Project).
			WithFreightNameOrAlias(nameOrAlias),
		nil,
	); err != nil {
		return client.FormatAPIError("unreject freight", err)
	}
	_, _ = fmt.Fprintln(o.Out, "Freight rejection cleared.")
	return nil
}
