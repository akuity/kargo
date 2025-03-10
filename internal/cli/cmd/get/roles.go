package get

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/io"
	"github.com/akuity/kargo/internal/cli/kubernetes"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
)

type getRolesOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	*getOptions

	Config        config.CLIConfig
	ClientOptions client.Options

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
	o.PrintFlags.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project for which to list roles. If not set, the default project will be used.",
	)

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
	// While the flags are marked as required, a user could still provide an empty
	// string. This is a check to ensure that the flags are not empty.
	if o.Project == "" {
		return fmt.Errorf("%s is required", option.ProjectFlag)
	}
	return nil
}

// run gets the the roles from the server and prints them to the console.
func (o *getRolesOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	var kargoRoleRes []*rbacapi.Role
	var resourcesRes []*rbacapi.RoleResources
	var errs []error

	if len(o.Names) == 0 {
		var resp *connect.Response[v1alpha1.ListRolesResponse]
		if resp, err = kargoSvcCli.ListRoles(
			ctx,
			connect.NewRequest(&v1alpha1.ListRolesRequest{
				Project:     o.Project,
				AsResources: o.AsKubernetesResources,
			}),
		); err != nil {
			return fmt.Errorf("list roles: %w", err)
		}
		if o.AsKubernetesResources {
			resourcesRes = make([]*rbacapi.RoleResources, len(resp.Msg.GetResources()))
			for i, roleResources := range resp.Msg.GetResources() {
				resourcesRes[i] = roleResources
			}
		} else {
			kargoRoleRes = resp.Msg.GetRoles()
		}
	} else {
		errs = make([]error, 0, len(o.Names))
		if o.AsKubernetesResources {
			resourcesRes = make([]*rbacapi.RoleResources, 0, len(o.Names))
		} else {
			kargoRoleRes = make([]*rbacapi.Role, 0, len(o.Names))
		}
		for _, name := range o.Names {
			var resp *connect.Response[v1alpha1.GetRoleResponse]
			if resp, err = kargoSvcCli.GetRole(
				ctx,
				connect.NewRequest(
					&v1alpha1.GetRoleRequest{
						Project:     o.Project,
						Name:        name,
						AsResources: o.AsKubernetesResources,
					},
				),
			); err != nil {
				errs = append(errs, err)
				continue
			}
			if o.AsKubernetesResources {
				resourcesRes = append(resourcesRes, resp.Msg.GetResources())
			} else {
				kargoRoleRes = append(kargoRoleRes, resp.Msg.GetRole())
			}
		}
	}

	if o.AsKubernetesResources {
		if err = printObjects(resourcesRes, o.PrintFlags, o.IOStreams, o.NoHeaders); err != nil {
			return fmt.Errorf("print resources: %w", err)
		}
	} else {
		if err = printObjects(kargoRoleRes, o.PrintFlags, o.IOStreams, o.NoHeaders); err != nil {
			return fmt.Errorf("print roles: %w", err)
		}
	}

	return errors.Join(errs...)
}

func newRoleTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		role := item.Object.(*rbacapi.Role) // nolint: forcetypeassert
		rows[i] = metav1.TableRow{
			Cells: []any{
				role.ObjectMeta.Name,
				role.KargoManaged,
				duration.HumanDuration(time.Since(role.ObjectMeta.CreationTimestamp.Time)),
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
