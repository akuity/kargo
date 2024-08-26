package grant

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/io"
	"github.com/akuity/kargo/internal/cli/kubernetes"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type grantOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Project      string
	Role         string
	Subs         []string
	Emails       []string
	Groups       []string
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
		Use: `grant [--project=project] --role=role \
		[--sub=sub] [--email=email] [--group=group] [--claim=name=value] \
		[--resource-type=resource-type [--resource-name=resource-name] --verb=verb]`,
		Short: "Grant a role to a user or grant permissions to a role",
		Args:  option.NoArgs,
		Example: templates.Example(`
# Grant my-role permission to update all stages
kargo grant --project=my-project --role=my-role --resource-type=stage --verb=update

# Grant my-role permission to promote to stage dev
kargo grant --project=my-project --role=my-role \
  --resource-type=stage --resource-name=dev --verb=promote

# Grant my-role to users with specific nonstandard claim, email address, groups and sub claim
kargo grant --project=my-project --role=my-role \
  --claim=given_name=bob --claim=emails=alice@example.com \
  --claim=subs=1234567890 --claim=groups=admins
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
	o.PrintFlags.AddFlags(cmd)
	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project in which to manage a role. If not set, the default project will be used.",
	)
	option.Role(cmd.Flags(), &o.Role, "The role to manage.")
	option.Claims(cmd.Flags(), &o.Claims, "A claim name and value to be granted to the role.")

	option.ResourceType(cmd.Flags(), &o.ResourceType, "A type of resource to grant permissions to.")
	option.ResourceName(cmd.Flags(), &o.ResourceName, "The name of a resource to grant permissions to.")
	option.Verbs(cmd.Flags(), &o.Verbs, "A verb to grant on the resource.")

	if err := cmd.MarkFlagRequired(option.RoleFlag); err != nil {
		panic(fmt.Errorf("could not mark %s flag as required: %w", option.RoleFlag, err))
	}

	// If none of these are specified, we're not granting anything.
	cmd.MarkFlagsOneRequired(
		option.ClaimFlag,
		option.ResourceTypeFlag,
	)

	// You can't grant a role to users and grant permissions to a role at the same
	// time.
	cmd.MarkFlagsMutuallyExclusive(option.SubFlag, option.ResourceTypeFlag)
	cmd.MarkFlagsMutuallyExclusive(option.EmailFlag, option.ResourceTypeFlag)
	cmd.MarkFlagsMutuallyExclusive(option.GroupFlag, option.ResourceTypeFlag)
	cmd.MarkFlagsMutuallyExclusive(option.ClaimFlag, option.ResourceTypeFlag)

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
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	req := &svcv1alpha1.GrantRequest{
		Project: o.Project,
		Role:    o.Role,
	}
	if o.ResourceType != "" {
		req.Request = &svcv1alpha1.GrantRequest_ResourceDetails{
			ResourceDetails: &rbacapi.ResourceDetails{
				ResourceType: o.ResourceType,
				ResourceName: o.ResourceName,
				Verbs:        o.Verbs,
			},
		}
	} else {
		claimsList := svcv1alpha1.ListUserClaims{}

		for _, claimFlagValue := range o.Claims {
			claimFlagNameAndValue := strings.Split(claimFlagValue, "=")
			claimsList.UserClaims = append(claimsList.UserClaims, &rbacapi.UserClaim{
				Name:   claimFlagNameAndValue[0],
				Values: []string{claimFlagNameAndValue[1]},
			})
		}
		req.Request = &svcv1alpha1.GrantRequest_UserClaims{
			UserClaims: &claimsList,
		}
	}

	resp, err := kargoSvcCli.Grant(ctx, connect.NewRequest(req))
	if err != nil {
		return fmt.Errorf("grant: %w", err)
	}

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return fmt.Errorf("new printer: %w", err)
	}

	return printer.PrintObj(resp.Msg.Role, o.IOStreams.Out)
}
