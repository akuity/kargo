package get

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
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

type getRolesOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	*getOptions

	Config        config.CLIConfig
	ClientOptions client.Options

	SystemLevel           bool
	Project               string
	Names                 []string
	AsKubernetesResources bool
}

func newRolesCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
	getOptions *getOptions,
) *cobra.Command {
	cmdOpts := &getRolesOptions{
		Config:     cfg,
		IOStreams:  streams,
		getOptions: getOptions,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "roles [--project=project] [NAME ...] [--no-headers]",
		Aliases: []string{"role"},
		Short:   "Display one or many roles",
		Example: templates.Example(`
# List all roles in my-project
kargo get roles --project=my-project

# List all roles in my-project in JSON output format
kargo get roles --project=my-project -o json

# Get the dev role in my-project
kargo get role --project=my-project dev

# List all roles in the default project
kargo config set-project my-project
kargo get roles

# Get a the dev role in the default project
kargo config set-project my-project
kargo get role dev

# List all system-level roles
kargo get roles --system

# Get the kargo-admin system-level role
kargo get role --system kargo-admin
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

// addFlags adds the flags for the get roles options to the provided command.
func (o *getRolesOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project for which to list roles. If not set, the default project will be used.",
	)
	option.System(
		cmd.Flags(), &o.SystemLevel, false,
		"Whether to list system-level roles instead of project-level roles",
	)
	// project and system flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SystemFlag)

	option.AsKubernetesResources(
		cmd.Flags(), &o.AsKubernetesResources,
		"Output the roles as Kubernetes resources.",
	)
}

// complete sets the options from the command arguments.
func (o *getRolesOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *getRolesOptions) validate() error {
	if o.Project == "" && !o.SystemLevel {
		return fmt.Errorf(
			"either %s or %s is required", option.ProjectFlag, option.SystemFlag,
		)
	}
	return nil
}

// run gets the the roles from the server and prints them to the console.
func (o *getRolesOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	// Note: The REST API does not currently support the AsKubernetesResources flag
	// If this flag is set, return an error indicating it's not supported
	if o.AsKubernetesResources {
		return fmt.Errorf("--as-k8s-resources flag is not supported with REST API")
	}

	var kargoRoleRes []*rbacapi.Role
	var errs []error

	if len(o.Names) == 0 {
		var payload any
		if o.SystemLevel {
			var res *rbac.ListSystemRolesOK
			if res, err = apiClient.Rbac.ListSystemRoles(
				rbac.NewListSystemRolesParams(),
				nil,
			); err != nil {
				return fmt.Errorf("list roles: %w", err)
			}
			payload = res.Payload
		} else {
			var res *rbac.ListProjectRolesOK
			if res, err = apiClient.Rbac.ListProjectRoles(
				rbac.NewListProjectRolesParams().WithProject(o.Project),
				nil,
			); err != nil {
				return fmt.Errorf("list roles: %w", err)
			}
			payload = res.Payload
		}

		var listJSON []byte
		if listJSON, err = json.Marshal(payload); err != nil {
			return err
		}
		if err = json.Unmarshal(listJSON, &kargoRoleRes); err != nil {
			return err
		}
	} else {
		errs = make([]error, 0, len(o.Names))
		kargoRoleRes = make([]*rbacapi.Role, 0, len(o.Names))

		for _, name := range o.Names {
			var payload any
			if o.SystemLevel {
				var res *rbac.GetSystemRoleOK
				if res, err = apiClient.Rbac.GetSystemRole(
					rbac.NewGetSystemRoleParams().WithRole(name),
					nil,
				); err != nil {
					errs = append(errs, err)
					continue
				}
				payload = res.Payload
			} else {
				var res *rbac.GetProjectRoleOK
				if res, err = apiClient.Rbac.GetProjectRole(
					rbac.NewGetProjectRoleParams().
						WithProject(o.Project).
						WithRole(name),
					nil,
				); err != nil {
					errs = append(errs, err)
					continue
				}
				payload = res.Payload
			}

			var roleJSON []byte
			if roleJSON, err = json.Marshal(payload); err != nil {
				errs = append(errs, err)
				continue
			}
			var role *rbacapi.Role
			if err = json.Unmarshal(roleJSON, &role); err != nil {
				errs = append(errs, err)
				continue
			}
			kargoRoleRes = append(kargoRoleRes, role)
		}
	}

	if err = PrintObjects(kargoRoleRes, o.PrintFlags, o.IOStreams, o.NoHeaders); err != nil {
		return fmt.Errorf("print roles: %w", err)
	}

	return errors.Join(errs...)
}

func newRoleTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		role := item.Object.(*rbacapi.Role) // nolint: forcetypeassert
		rows[i] = metav1.TableRow{
			Cells: []any{
				role.Name,
				role.KargoManaged,
				duration.HumanDuration(time.Since(role.CreationTimestamp.Time)),
			},
			Object: list.Items[i],
		}
	}
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Kargo Managed", Type: "bool"},
			{Name: "Age", Type: "string"},
		},
		Rows: rows,
	}
}

func newRoleResourcesTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		roleResources := item.Object.(*rbacapi.RoleResources) // nolint: forcetypeassert
		rbs := make([]string, len(roleResources.RoleBindings))
		for i, rb := range roleResources.RoleBindings {
			rbs[i] = rb.Name
		}
		roles := make([]string, len(roleResources.Roles))
		for i, role := range roleResources.Roles {
			roles[i] = role.Name
		}
		rows[i] = metav1.TableRow{
			Cells: []any{
				roleResources.ServiceAccount.Name,
				roleResources.ServiceAccount.Name,
				strings.Join(rbs, ", "),
				strings.Join(roles, ", "),
				duration.HumanDuration(time.Since(roleResources.ServiceAccount.CreationTimestamp.Time)),
			},
			Object: list.Items[i],
		}
	}
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "k8s Service Account", Type: "string"},
			{Name: "k8s Role Bindings", Type: "string"},
			{Name: "k8s Roles", Type: "string"},
			{Name: "Age", Type: "string"},
		},
		Rows: rows,
	}
}
