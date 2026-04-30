package grant

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

type grantOptions struct {
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
	cmdOpts := &grantOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("updated").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use: `grant [--project=project] --role=role [--claim=name=value]... \
		[--verb=verb --resource-type=resource-type [--resource-name=resource-name]]`,
		Short: "Grant a role to a user or grant permissions to a role",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Grant my-role permission to update all stages
kargo grant --project=my-project --role=my-role \
  --verb=update --resource-type=stage

# Grant my-role permission to promote to stage dev
kargo grant --project=my-project --role=my-role \
  --verb=promote --resource-type=stage --resource-name=dev

# Grant my-role to users with specific claims
kargo grant --project=my-project --role=my-role \
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

// addFlags adds the flags for the grant options to the provided command.
func (o *grantOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)
	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project in which to manage a role. If not set, the default project will be used.",
	)
	option.Role(cmd.Flags(), &o.Role, "The role to manage.")
	option.Claims(cmd.Flags(), &o.Claims, "An OIDC claim name and value.")

	option.ResourceType(cmd.Flags(), &o.ResourceType, "A type of resource to grant permissions to.")
	option.ResourceName(cmd.Flags(), &o.ResourceName, "The name of a resource to grant permissions to.")
	option.Verbs(cmd.Flags(), &o.Verbs, "A verb to grant on the resource.")

	if err := cmd.MarkFlagRequired(option.RoleFlag); err != nil {
		panic(fmt.Errorf("could not mark %s flag as required: %w", option.RoleFlag, err))
	}

	// If none of these are specified, we're not granting anything.
	cmd.MarkFlagsOneRequired(option.ClaimFlag, option.ResourceTypeFlag)

	// You can't grant a role to users and grant permissions to a role at the same
	// time.
	cmd.MarkFlagsMutuallyExclusive(option.ClaimFlag, option.VerbFlag)
	cmd.MarkFlagsMutuallyExclusive(option.ClaimFlag, option.ResourceTypeFlag)
	cmd.MarkFlagsMutuallyExclusive(option.ClaimFlag, option.ResourceNameFlag)

	cmd.MarkFlagsRequiredTogether(option.ResourceTypeFlag, option.VerbFlag)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *grantOptions) validate() error {
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

// run grants a role to users or grants permissions to a role.
func (o *grantOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	req := &models.GrantRequest{
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

	_, err = apiClient.Rbac.Grant(
		rbac.NewGrantParams().WithProject(o.Project).WithBody(req),
		nil,
	)
	if err != nil {
		return fmt.Errorf("grant: %w", err)
	}

	// Get the updated role after granting
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
