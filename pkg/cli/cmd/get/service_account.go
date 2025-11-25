package get

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	rbacapi "github.com/akuity/kargo/api/rbac/v1alpha1"
	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
)

type getServiceAccountsOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	*getOptions

	Config        config.CLIConfig
	ClientOptions client.Options

	SystemLevel bool
	Project     string
	Names       []string
}

func newGetServiceAccountsCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
	getOptions *getOptions,
) *cobra.Command {
	cmdOpts := &getServiceAccountsOptions{
		Config:     cfg,
		IOStreams:  streams,
		getOptions: getOptions,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "serviceaccount [--project=project] [NAME ...] [--no-headers]",
		Aliases: []string{"serviceaccounts", "sa", "sas"},
		Short:   "Display one or many service accounts",
		Example: templates.Example(`
# List all service accounts in my-project
kargo get serviceaccounts --project=my-project

# List all service accounts in my-project in JSON output format
kargo get serviceaccounts --project=my-project -o json

# Get the my-service-account service account in my-project
kargo get serviceaccount --project=my-project \
  my-service-account

# List all service accounts in the default project
kargo config set-project my-project
kargo get serviceaccounts

# Get a service account in the default project
kargo config set-project my-project
kargo get serviceaccount my-service-account

# List system-level service accounts
kargo get serviceaccounts --system

# Get the system-level kargo-admin service account
kargo get serviceaccount --system kargo-admin
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

// addFlags adds the flags for the get service accounts options to the provided
// command.
func (o *getServiceAccountsOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project for which to list service accounts. If not set, the default "+
			"project will be used.",
	)
	option.System(
		cmd.Flags(), &o.SystemLevel, false,
		"Whether to list system-level service accounts instead of project-level "+
			"service accounts.",
	)
	// project and system flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SystemFlag)
}

// complete sets the options from the command arguments.
func (o *getServiceAccountsOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *getServiceAccountsOptions) validate() error {
	if o.Project == "" && !o.SystemLevel {
		return fmt.Errorf(
			"either %s or %s is required", option.ProjectFlag, option.SystemFlag,
		)
	}
	return nil
}

// run gets the the service accounts from the server and prints them to the
// console.
func (o *getServiceAccountsOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	if len(o.Names) == 0 {
		var resp *connect.Response[v1alpha1.ListServiceAccountsResponse]
		if resp, err = kargoSvcCli.ListServiceAccounts(
			ctx,
			connect.NewRequest(
				&v1alpha1.ListServiceAccountsRequest{
					SystemLevel: o.SystemLevel,
					Project:     o.Project,
				},
			),
		); err != nil {
			return fmt.Errorf("list service accounts: %w", err)
		}
		return PrintObjects(
			resp.Msg.GetServiceAccounts(),
			o.PrintFlags,
			o.IOStreams,
			o.NoHeaders,
		)
	}

	res := make([]*corev1.ServiceAccount, 0, len(o.Names))
	errs := make([]error, 0, len(o.Names))
	for _, name := range o.Names {
		var resp *connect.Response[v1alpha1.GetServiceAccountResponse]
		if resp, err = kargoSvcCli.GetServiceAccount(
			ctx,
			connect.NewRequest(
				&v1alpha1.GetServiceAccountRequest{
					SystemLevel: o.SystemLevel,
					Project:     o.Project,
					Name:        name,
				},
			),
		); err != nil {
			errs = append(errs, err)
			continue
		}
		res = append(res, resp.Msg.GetServiceAccount())
	}

	if err = PrintObjects(res, o.PrintFlags, o.IOStreams, o.NoHeaders); err != nil {
		return fmt.Errorf("print service accounts: %w", err)
	}
	return errors.Join(errs...)
}

func newServiceAccountTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		sa := item.Object.(*corev1.ServiceAccount) // nolint: forcetypeassert
		rows[i] = metav1.TableRow{
			Cells: []any{
				sa.Name,
				sa.Annotations[rbacapi.AnnotationKeyManaged] == rbacapi.AnnotationValueTrue,
				duration.HumanDuration(time.Since(sa.CreationTimestamp.Time)),
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
