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

	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/io"
	"github.com/akuity/kargo/internal/cli/kubernetes"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
	libCreds "github.com/akuity/kargo/internal/credentials"
)

type getCredentialsOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	*getOptions

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
	Names   []string
}

func newGetCredentialsCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
	getOptions *getOptions,
) *cobra.Command {
	cmdOpts := &getCredentialsOptions{
		Config:     cfg,
		IOStreams:  streams,
		getOptions: getOptions,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use:     "credentials [--project=project] [NAME ...] [--no-headers]",
		Aliases: []string{"credential", "creds", "cred"},
		Short:   "Display one or many credentials",
		Example: templates.Example(`
# List all credentials in my-project
kargo get credentials --project=my-project

# Get specific credentials in my-project
kargo get credentials --project=my-project my-credentials

# List all credentials in the default project
kargo config set-project my-project
kargo get credentials

# Get specific credentials in the default project
kargo config set-project my-project
kargo get credentials my-credentials`),
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

// addFlags adds the flags for the get credentials options to the provided
// command.
func (o *getCredentialsOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.PrintFlags.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project for which to list credentials. If not set, the default project will be used.",
	)
}

// complete sets the options from the command arguments.
func (o *getCredentialsOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *getCredentialsOptions) validate() error {
	// While the flags are marked as required, a user could still provide an empty
	// string. This is a check to ensure that the flags are not empty.
	if o.Project == "" {
		return errors.New("project is required")
	}
	return nil
}

// run gets the credentials from the server and prints them to the console.
func (o *getCredentialsOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	if len(o.Names) == 0 {
		var resp *connect.Response[v1alpha1.ListCredentialsResponse]
		if resp, err = kargoSvcCli.ListCredentials(
			ctx,
			connect.NewRequest(
				&v1alpha1.ListCredentialsRequest{
					Project: o.Project,
				},
			),
		); err != nil {
			return fmt.Errorf("list credentials: %w", err)
		}
		return printObjects(resp.Msg.GetCredentials(), o.PrintFlags, o.IOStreams, o.NoHeaders)
	}

	res := make([]*corev1.Secret, 0, len(o.Names))
	errs := make([]error, 0, len(o.Names))
	for _, name := range o.Names {
		var resp *connect.Response[v1alpha1.GetCredentialsResponse]
		if resp, err = kargoSvcCli.GetCredentials(
			ctx,
			connect.NewRequest(
				&v1alpha1.GetCredentialsRequest{
					Project: o.Project,
					Name:    name,
				},
			),
		); err != nil {
			errs = append(errs, err)
			continue
		}
		res = append(res, resp.Msg.GetCredentials())
	}

	if err = printObjects(res, o.PrintFlags, o.IOStreams, o.NoHeaders); err != nil {
		return fmt.Errorf("print stages: %w", err)
	}
	return errors.Join(errs...)
}

func newCredentialsTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		secret := item.Object.(*corev1.Secret) // nolint: forcetypeassert
		rows[i] = metav1.TableRow{
			Cells: []any{
				secret.Name,
				secret.ObjectMeta.Labels[kargoapi.CredentialTypeLabelKey],
				secret.StringData[libCreds.FieldRepoURLIsRegex],
				secret.StringData[libCreds.FieldRepoURL],
				duration.HumanDuration(time.Since(secret.CreationTimestamp.Time)),
			},
			Object: list.Items[i],
		}
	}
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Type", Type: "string"},
			{Name: "Regex", Type: "string"},
			{Name: "Repo", Type: "string"},
			{Name: "Age", Type: "string"},
		},
		Rows: rows,
	}
}
