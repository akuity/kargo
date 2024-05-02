package create

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/io"
	"github.com/akuity/kargo/internal/cli/kubernetes"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
	kargosvcapi "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type createRoleOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Project     string
	Name        string
	Description string
	Subs        []string
	Emails      []string
	Groups      []string
}

func newRoleCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &createRoleOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("created").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:   "role [--project=project] NAME [--sub=subject] [--email=email] [--group]",
		Short: "Create a role",
		Args:  option.ExactArgs(1),
		Example: templates.Example(`
# Create a role in a project without initially granting it to any users
kargo create role --project=my-project my-role

# Create a role in a project and grant it to users with specific sub claims
kargo create role --project=my-project my-role \
  --sub=1234567890 --sub=0987654321

# Create a role in a project and grant it to users with specific email addresses
kargo create role --project=my-project my-role \
  --email=bob@example.com --email=alice@example.com

# Create a role in a project and grant it to users in specific groups
kargo create role --project=my-project my-role \
  --group=admins --group=engineers

# Create a role the default project without initially granting it to any users
kargo config set-project my-project
kargo create role my-role

# Create a role in the default project and grant it to users with specific sub claims
kargo config set-project my-project
kargo create role my-role \
  --sub=1234567890 --sub=0987654321

# Create a role in the default project and grant it to users with specific email addresses
kargo config set-project my-project
kargo create role my-role \
  --email=bob@example.com --email=alice@example.com

# Create a role in the default project and grant it to users in specific groups
kargo config set-project my-project
kargo create role my-role \
  --group=admins --group=engineers
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

// addFlags adds the flags for the create role options to the provided command.
func (o *createRoleOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.PrintFlags.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project in which to create the role. If not set, the default project will be used.",
	)
	option.Description(cmd.Flags(), &o.Description, "Description of the role.")
	option.Subs(cmd.Flags(), &o.Subs, "A subject claim to map to the role.")
	option.Emails(cmd.Flags(), &o.Emails, "An email address to map to the role.")
	option.Groups(cmd.Flags(), &o.Groups, "A group claim to map to the role.")
}

// complete sets the options from the command arguments.
func (o *createRoleOptions) complete(args []string) {
	o.Name = strings.TrimSpace(args[0])
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *createRoleOptions) validate() error {
	var errs []error
	// While the flags are marked as required, a user could still provide an empty
	// string. This is a check to ensure that the flags are not empty.
	if o.Project == "" {
		errs = append(errs, fmt.Errorf("%s is required", option.ProjectFlag))
	}
	if o.Name == "" {
		errs = append(errs, fmt.Errorf("%s is required", option.NameFlag))
	}
	return errors.Join(errs...)
}

// run creates a role using the provided options.
func (o *createRoleOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	resp, err := kargoSvcCli.CreateRole(
		ctx,
		connect.NewRequest(
			&kargosvcapi.CreateRoleRequest{
				Role: &rbacapi.Role{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: o.Project,
						Name:      o.Name,
					},
					Subs:   o.Subs,
					Emails: o.Emails,
					Groups: o.Groups,
				},
			},
		),
	)
	if err != nil {
		return fmt.Errorf("create role: %w", err)
	}

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return fmt.Errorf("new printer: %w", err)
	}

	return printer.PrintObj(resp.Msg.Role, o.IOStreams.Out)
}
