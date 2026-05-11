package revoke

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/models"
	"github.com/akuity/kargo/pkg/client/generated/rbac"
)

type revokeOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Project      string
	Role         string
	Claims       []string
	ResourceType string
	ResourceName string
	Verbs        []string
}

func NewCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &revokeOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("updated").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use: `revoke [--project=project] --role=role [--claim=name=value]... \
		[--verb=verb --resource-type=resource-type [--resource-name=resource-name]]`,
		Short: "Revoke a role from a user or revoke permissions from a role",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Revoke permission to update all stages from my-role
kargo revoke --project=my-project --role=my-role \
  --verb=update --resource-type=stage

# Revoke permission to promote to stage dev from my-role
kargo revoke --project=my-project --role=my-role \
  --verb=promote --resource-type=stage --resource-name=dev

# Revoke my-role from users with specific claims
kargo revoke --project=my-project --role=my-role \
  --claim=email=alice@example.com --claim=groups=admins,power-users
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

// addFlags adds the flags for the revoke options to the provided command.
func (o *revokeOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)
	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project in which to manage a role. If not set, the default project will be used.",
	)
	option.Role(cmd.Flags(), &o.Role, "The role to manage.")
	option.Claims(cmd.Flags(), &o.Claims, "A OIDC claim name and value.")

	option.ResourceType(cmd.Flags(), &o.ResourceType, "A type of resource to revoke permissions for.")
	option.ResourceName(cmd.Flags(), &o.ResourceName, "The name of a resource to revoke permissions for.")
	option.Verbs(cmd.Flags(), &o.Verbs, "A verb to revoke on the resource.")

	if err := cmd.MarkFlagRequired(option.RoleFlag); err != nil {
		panic(fmt.Errorf("could not mark %s flag as required: %w", option.RoleFlag, err))
	}

	// If none of these are specified, we're not revoking anything.
	cmd.MarkFlagsOneRequired(option.ClaimFlag, option.ResourceTypeFlag)

	// You can't revoke a role from users and revoke permissions from a role at
	// the same time.
	cmd.MarkFlagsMutuallyExclusive(option.ClaimFlag, option.VerbFlag)
	cmd.MarkFlagsMutuallyExclusive(option.ClaimFlag, option.ResourceTypeFlag)
	cmd.MarkFlagsMutuallyExclusive(option.ClaimFlag, option.ResourceNameFlag)

	cmd.MarkFlagsRequiredTogether(option.ResourceTypeFlag, option.VerbFlag)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *revokeOptions) validate() error {
	var errs []error
	// While the flags are marked as required, a user could still provide an empty
	// string. This is a check to ensure that the flags are not empty.
	if o.Project == "" {
		errs = append(errs, fmt.Errorf("%s is required", option.ProjectFlag))
	}
	if o.Role == "" {
		errs = append(errs, fmt.Errorf("%s is required", option.RoleFlag))
	}
	// This is a check to ensure that any claims flags have exactly 1 "=".
	for _, claim := range o.Claims {
		if strings.Count(claim, "=") != 1 {
			errs = append(errs, fmt.Errorf("%s should be in the format <claim-name>=<claim-value>", option.ClaimFlag))
		}
	}
	return errors.Join(errs...)
}

// run revokes a role from users or revokes permissions from a role.
func (o *revokeOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	req := &models.RevokeRequest{
		Role: o.Role,
	}
	if o.ResourceType != "" {
		req.ResourceDetails = &models.ResourceDetails{
			ResourceType: o.ResourceType,
			ResourceName: o.ResourceName,
			Verbs:        o.Verbs,
		}
	} else {
		claims := make([]*models.Claim, 0, len(o.Claims))
		for _, claimFlagValue := range o.Claims {
			claimFlagNameAndValue := strings.Split(claimFlagValue, "=")
			claims = append(claims, &models.Claim{
				Name:   claimFlagNameAndValue[0],
				Values: []string{claimFlagNameAndValue[1]},
			})
		}
		req.UserClaims = &models.UserClaims{
			Claims: claims,
		}
	}

	_, err = apiClient.Rbac.Revoke(
		rbac.NewRevokeParams().WithProject(o.Project).WithBody(req),
		nil,
	)
	if err != nil {
		return fmt.Errorf("revoke: %w", err)
	}

	// Get the updated role after revocation
	res, err := apiClient.Rbac.GetProjectRole(
		rbac.NewGetProjectRoleParams().WithProject(o.Project).WithRole(o.Role),
		nil,
	)
	if err != nil {
		return fmt.Errorf("get role: %w", err)
	}

	roleJSON, err := json.Marshal(res.Payload)
	if err != nil {
		return fmt.Errorf("marshal role: %w", err)
	}
	var role *rbacapi.Role
	if err = json.Unmarshal(roleJSON, &role); err != nil {
		return fmt.Errorf("unmarshal role: %w", err)
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("new printer: %w", err)
	}

	return printer.PrintObj(role, o.Out)
}
