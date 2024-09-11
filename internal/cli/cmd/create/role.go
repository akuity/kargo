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
	Claims      []string
}

func newRoleCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &createRoleOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("created").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:   "role [--project=project] NAME [--claim=name=value1,value2,...]...",
		Short: "Create a role",
		Args:  option.ExactArgs(1),
		Example: templates.Example(`
# Create a role in a project without initially granting it to any users
kargo create role --project=my-project my-role

# Create a role in a project and grant it to users with specific claims
kargo create role --project=my-project my-role \
  --claim=email=alice@example.com --claim=groups=admins,power-users

# Create a role the default project without initially granting it to any users
kargo config set-project my-project
kargo create role my-role

# Create a role in the default project and grant it to users with specific claims
kargo config set-project my-project
kargo create role my-role \
  --claim=email=alice@example.com --claim=groups=admins,power-users
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
	option.Claims(cmd.Flags(), &o.Claims, "A claim name and value to map to the role")
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
	// This is a check to ensure that any claims flags have exactly 1 "=".
	for _, claim := range o.Claims {
		if strings.Count(claim, "=") != 1 {
			errs = append(errs, fmt.Errorf("%s should be in the format <claim-name>=<claim-value>", option.ClaimFlag))
		}
	}
	return errors.Join(errs...)
}

// run creates a role using the provided options.
func (o *createRoleOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	claims := []*rbacapi.UserClaim{}

	for _, claimFlagValue := range o.Claims {
		claimFlagNameAndValue := strings.Split(claimFlagValue, "=")
		claims = append(claims, &rbacapi.UserClaim{
			Name:   claimFlagNameAndValue[0],
			Values: []string{claimFlagNameAndValue[1]},
		})
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
					Claims: claims,
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
