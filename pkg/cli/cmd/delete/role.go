package delete

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/rbac"
)

type deleteRoleOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
	Names   []string
}

func newRoleCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &deleteRoleOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("deleted").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:   "role [--project=project] (NAME ...)",
		Short: "Delete role by name",
		Args:  option.MinimumNArgs(1),
		Example: templates.Example(`
# Delete a role
kargo delete role --project=my-project my-role

# Delete multiple roles
kargo delete role --project=my-project my-role1 my-role2

# Delete a role in the default project
kargo config set-project my-project
kargo delete role my-role
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdOpts.complete(args)

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

// addFlags adds the flags for the delete role options to the provided command.
func (o *deleteRoleOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(cmd.Flags(), &o.Project, o.Config.Project,
		"The Project for which to delete Roles. If not set, the default project will be used.")
}

// complete sets the options from the command arguments.
func (o *deleteRoleOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *deleteRoleOptions) validate() error {
	var errs []error
	// While the flags are marked as required, a user could still provide an empty
	// string. This is a check to ensure that the flags are not empty.
	if o.Project == "" {
		errs = append(errs, fmt.Errorf("%s is required", option.ProjectFlag))
	}
	if len(o.Names) == 0 {
		errs = append(errs, fmt.Errorf("%s is required", option.NameFlag))
	}
	return errors.Join(errs...)
}

// run removes the role(s) from the project based on the options.
func (o *deleteRoleOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("create printer: %w", err)
	}

	var errs []error
	for _, name := range o.Names {
		if _, err := apiClient.Rbac.DeleteProjectRole(
			rbac.NewDeleteProjectRoleParams().
				WithProject(o.Project).
				WithRole(name),
			nil,
		); err != nil {
			errs = append(errs, err)
			continue
		}
		_ = printer.PrintObj(
			&rbacapi.Role{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: o.Project,
					Name:      name,
				},
			},
			o.Out,
		)
	}
	return errors.Join(errs...)
}
