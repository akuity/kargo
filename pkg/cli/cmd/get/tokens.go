package get

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
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

type getTokensOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	*getOptions

	Config        config.CLIConfig
	ClientOptions client.Options

	SystemLevel bool
	Project     string
	RoleName    string
	Names       []string
}

func newGetTokensCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
	getOptions *getOptions,
) *cobra.Command {
	cmdOpts := &getTokensOptions{
		Config:     cfg,
		IOStreams:  streams,
		getOptions: getOptions,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "token [--project=project] [--role=role] [NAME ...] [--no-headers]",
		Aliases: []string{"tokens"},
		Short:   "List tokens associated with a service account",
		Example: templates.Example(`
# Get the token named my-token in my-project
kargo get token --project=my-project my-token

# List all tokens for role my-role in my-project
kargo get token --project=my-project --role=my-role

# List all tokens for role my-role in my-project in JSON
# output format
kargo get tokens --project=my-project --role=my-role -o json

# List all tokens for role my-role in the default project
kargo config set-project my-project
kargo get tokens --role=my-role

# List all tokens for system-level roles
kargo get tokens --system

# List all tokens for the system-level kargo-admin role
kargo get tokens --system --role=kargo-admin
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

// addFlags adds the flags for the get API tokens options to the provided
// command.
func (o *getTokensOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project for which to list API tokens. If not set, the default "+
			"project will be used.",
	)
	option.System(
		cmd.Flags(), &o.SystemLevel, false,
		"Whether to list tokens for system-level role instead of project-level "+
			"roles.",
	)
	// project and system flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SystemFlag)

	option.Role(
		cmd.Flags(),
		&o.RoleName,
		"The role for which to list tokens. If not set, tokens for all roles in "+
			"the project (or system, if --system is set) will be listed.",
	)
}

// complete sets the options from the command arguments.
func (o *getTokensOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *getTokensOptions) validate() error {
	if o.Project == "" && !o.SystemLevel {
		return fmt.Errorf(
			"either %s or %s is required", option.ProjectFlag, option.SystemFlag,
		)
	}
	return nil
}

// run gets the tokens from the server and prints them to the console.
func (o *getTokensOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	if len(o.Names) == 0 {
		var payload any
		if o.SystemLevel {
			params := rbac.NewListSystemAPITokensParams()
			if o.RoleName != "" {
				params = params.WithRole(&o.RoleName)
			}
			var res *rbac.ListSystemAPITokensOK
			if res, err = apiClient.Rbac.ListSystemAPITokens(params, nil); err != nil {
				return fmt.Errorf("list API tokens: %w", err)
			}
			payload = res.GetPayload()
		} else {
			params := rbac.NewListProjectAPITokensParams().WithProject(o.Project)
			if o.RoleName != "" {
				params = params.WithRole(&o.RoleName)
			}
			var res *rbac.ListProjectAPITokensOK
			if res, err = apiClient.Rbac.ListProjectAPITokens(params, nil); err != nil {
				return fmt.Errorf("list API tokens: %w", err)
			}
			payload = res.GetPayload()
		}

		var listJSON []byte
		if listJSON, err = json.Marshal(payload); err != nil {
			return fmt.Errorf("marshal response: %w", err)
		}
		var secretList corev1.SecretList
		if err = json.Unmarshal(listJSON, &secretList); err != nil {
			return fmt.Errorf("unmarshal secret list: %w", err)
		}

		secrets := make([]*corev1.Secret, len(secretList.Items))
		for i := range secretList.Items {
			secrets[i] = &secretList.Items[i]
		}
		return PrintObjects(secrets, o.PrintFlags, o.IOStreams, o.NoHeaders)
	}

	res := make([]*corev1.Secret, 0, len(o.Names))
	errs := make([]error, 0, len(o.Names))
	for _, name := range o.Names {
		var payload any
		if o.SystemLevel {
			var res *rbac.GetSystemAPITokenOK
			if res, err = apiClient.Rbac.GetSystemAPIToken(
				rbac.NewGetSystemAPITokenParams().WithApitoken(name),
				nil,
			); err != nil {
				errs = append(errs, err)
				continue
			}
			payload = res.GetPayload()
		} else {
			var res *rbac.GetProjectAPITokenOK
			if res, err = apiClient.Rbac.GetProjectAPIToken(
				rbac.NewGetProjectAPITokenParams().
					WithProject(o.Project).
					WithApitoken(name),
				nil,
			); err != nil {
				errs = append(errs, err)
				continue
			}
			payload = res.GetPayload()
		}

		var secretJSON []byte
		secretJSON, err = json.Marshal(payload)
		if err != nil {
			errs = append(errs, fmt.Errorf("marshal response: %w", err))
			continue
		}
		var secret *corev1.Secret
		if err = json.Unmarshal(secretJSON, &secret); err != nil {
			errs = append(errs, fmt.Errorf("unmarshal secret: %w", err))
			continue
		}
		res = append(res, secret)
	}

	if err = PrintObjects(res, o.PrintFlags, o.IOStreams, o.NoHeaders); err != nil {
		return fmt.Errorf("print API tokens: %w", err)
	}
	return errors.Join(errs...)
}

func newAPITokensTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		tokenSecret := item.Object.(*corev1.Secret) // nolint: forcetypeassert
		rows[i] = metav1.TableRow{
			Cells: []any{
				tokenSecret.Name,
				tokenSecret.Annotations["kubernetes.io/service-account.name"],
				tokenSecret.Annotations[rbacapi.AnnotationKeyManaged] == rbacapi.AnnotationValueTrue,
				duration.HumanDuration(time.Since(tokenSecret.CreationTimestamp.Time)),
			},
			Object: list.Items[i],
		}
	}
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Role", Type: "string"},
			{Name: "Kargo Managed", Type: "bool"},
			{Name: "Age", Type: "string"},
		},
		Rows: rows,
	}
}
