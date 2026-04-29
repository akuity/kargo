package delete

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
)

type deleteFreightOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
	Names   []string
	Aliases []string
}

func newFreightCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &deleteFreightOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("deleted").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:   "freight [--project=project] [--name=name] [--alias=alias]",
		Short: "Delete freight by name or alias",
		Args:  cobra.NoArgs,
		Example: templates.Example(`
# Delete a piece of freight by name
kargo delete freight --project=my-project --name=abc123

# Delete a piece of freight by alias
kargo delete freight --project=my-project --alias=my-alias

# Delete multiple pieces of freight by name
kargo delete freight --project=my-project --name=abc123 --name=def456

# Delete a piece of freight in the default project
kargo config set-project my-project
kargo delete freight --name=abc123
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

// addFlags adds the flags for the delete freight options to the provided command.
func (o *deleteFreightOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(cmd.Flags(), &o.Project, o.Config.Project,
		"The Project for which to delete Freight. If not set, the default project will be used.")
	option.Names(cmd.Flags(), &o.Names, "Name of a piece of freight to delete.")
	option.Aliases(cmd.Flags(), &o.Aliases, "Alias of a piece of freight to delete.")
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *deleteFreightOptions) validate() error {
	// While the flags are marked as required, a user could still provide an empty
	// string. This is a check to ensure that the flags are not empty.
	if o.Project == "" {
		return fmt.Errorf("%s is required", option.ProjectFlag)
	}

	if len(o.Names) == 0 && len(o.Aliases) == 0 {
		return errors.New("name or alias is required")
	}

	return nil
}

// run removes the freight from the project based on the options.
func (o *deleteFreightOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("create printer: %w", err)
	}

	var errs []error
	for _, nameOrAlias := range append(o.Names, o.Aliases...) {
		if _, err := apiClient.Core.DeleteFreight(
			core.NewDeleteFreightParams().
				WithProject(o.Project).
				WithFreightNameOrAlias(nameOrAlias),
			nil,
		); err != nil {
			errs = append(errs, err)
			continue
		}
		_ = printer.PrintObj(&kargoapi.Freight{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nameOrAlias,
				Namespace: o.Project,
			},
		}, o.Out)
	}
	return errors.Join(errs...)
}
