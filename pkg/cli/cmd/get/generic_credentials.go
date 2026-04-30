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

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/credentials"
)

type getGenericCredentialsOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	*getOptions

	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
	Shared  bool
	System  bool
	Names   []string
}

func newGetGenericCredentialsCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
	getOptions *getOptions,
) *cobra.Command {
	cmdOpts := &getGenericCredentialsOptions{
		Config:     cfg,
		IOStreams:  streams,
		getOptions: getOptions,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use: "generic-credentials [--project=project | --shared | --system] [NAME ...] [--no-headers]",
		Aliases: []string{
			"generic-credential",
			"generic-creds",
			"generic-cred",
			"genericcredentials",
			"genericcredential",
			"genericcreds",
			"genericcred",
		},
		Short: "Display one or many generic credentials",
		Example: templates.Example(`
# List all generic credentials in my-project
kargo get generic-credentials --project=my-project

# Get specific generic credentials in my-project
kargo get generic-credentials --project=my-project my-credentials

# List all generic credentials in the default project
kargo config set-project my-project
kargo get generic-credentials

# Get specific generic credentials in the default project
kargo config set-project my-project
kargo get generic-credentials my-credentials

# List shared generic credentials
kargo get generic-credentials --shared

# Get specific shared generic credentials
kargo get generic-credentials --shared my-credentials

# List system generic credentials
kargo get generic-credentials --system

# Get specific system generic credentials
kargo get generic-credentials --system my-credentials
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

// addFlags adds the flags for the get generic-credentials options to the provided
// command.
func (o *getGenericCredentialsOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project for which to list credentials. If not set, the default project will be used.",
	)
	option.Shared(
		cmd.Flags(), &o.Shared, false,
		"Whether to list shared credentials instead of project-specific credentials.",
	)
	option.System(
		cmd.Flags(), &o.System, false,
		"Whether to list system credentials instead of project-specific credentials.",
	)
	// project, shared, and system flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SharedFlag, option.SystemFlag)
}

// complete sets the options from the command arguments.
func (o *getGenericCredentialsOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *getGenericCredentialsOptions) validate() error {
	var errs []error
	if o.Project == "" && !o.Shared && !o.System {
		errs = append(errs, fmt.Errorf(
			"one of %s, %s, or %s is required",
			option.ProjectFlag, option.SharedFlag, option.SystemFlag,
		))
	}
	return errors.Join(errs...)
}

// run gets the credentials from the server and prints them to the console.
func (o *getGenericCredentialsOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	if len(o.Names) == 0 {
		var payload any
		switch {
		case o.System:
			var res *credentials.ListSystemGenericCredentialsOK
			if res, err = apiClient.Credentials.ListSystemGenericCredentials(
				credentials.NewListSystemGenericCredentialsParams(),
				nil,
			); err != nil {
				return fmt.Errorf("list credentials: %w", err)
			}
			payload = res.Payload
		case o.Shared:
			var res *credentials.ListSharedGenericCredentialsOK
			if res, err = apiClient.Credentials.ListSharedGenericCredentials(
				credentials.NewListSharedGenericCredentialsParams(),
				nil,
			); err != nil {
				return fmt.Errorf("list credentials: %w", err)
			}
			payload = res.Payload
		default:
			var res *credentials.ListProjectGenericCredentialsOK
			if res, err = apiClient.Credentials.ListProjectGenericCredentials(
				credentials.NewListProjectGenericCredentialsParams().WithProject(o.Project),
				nil,
			); err != nil {
				return fmt.Errorf("list credentials: %w", err)
			}
			payload = res.Payload
		}
		var credsJSON []byte
		if credsJSON, err = json.Marshal(payload); err != nil {
			return err
		}
		creds := struct {
			Items []*corev1.Secret `json:"items"`
		}{}
		if err = json.Unmarshal(credsJSON, &creds); err != nil {
			return err
		}
		return PrintGenericCredentials(creds.Items, o.PrintFlags, o.IOStreams, o.NoHeaders)
	}

	secrets := make([]*corev1.Secret, 0, len(o.Names))
	errs := make([]error, 0, len(o.Names))
	for _, name := range o.Names {
		var payload any
		switch {
		case o.System:
			var res *credentials.GetSystemGenericCredentialsOK
			if res, err = apiClient.Credentials.GetSystemGenericCredentials(
				credentials.NewGetSystemGenericCredentialsParams().
					WithGenericCredentials(name),
				nil,
			); err != nil {
				errs = append(errs, err)
				continue
			}
			payload = res.Payload
		case o.Shared:
			var res *credentials.GetSharedGenericCredentialsOK
			if res, err = apiClient.Credentials.GetSharedGenericCredentials(
				credentials.NewGetSharedGenericCredentialsParams().
					WithGenericCredentials(name),
				nil,
			); err != nil {
				errs = append(errs, err)
				continue
			}
			payload = res.Payload
		default:
			var res *credentials.GetProjectGenericCredentialsOK
			if res, err = apiClient.Credentials.GetProjectGenericCredentials(
				credentials.NewGetProjectGenericCredentialsParams().
					WithProject(o.Project).
					WithGenericCredentials(name),
				nil,
			); err != nil {
				errs = append(errs, err)
				continue
			}
			payload = res.Payload
		}
		var credJSON []byte
		if credJSON, err = json.Marshal(payload); err != nil {
			errs = append(errs, err)
			continue
		}
		var cred *corev1.Secret
		if err = json.Unmarshal(credJSON, &cred); err != nil {
			errs = append(errs, err)
			continue
		}
		secrets = append(secrets, cred)
	}

	if err = PrintGenericCredentials(secrets, o.PrintFlags, o.IOStreams, o.NoHeaders); err != nil {
		return fmt.Errorf("print credentials: %w", err)
	}
	return errors.Join(errs...)
}

// PrintGenericCredentials prints generic credentials to the output stream.
func PrintGenericCredentials(
	secrets []*corev1.Secret,
	flags *genericclioptions.PrintFlags,
	streams genericiooptions.IOStreams,
	noHeaders bool,
) error {
	return PrintObjects(secrets, flags, streams, noHeaders)
}

func newGenericCredentialsTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		secret := item.Object.(*corev1.Secret) // nolint: forcetypeassert
		rows[i] = metav1.TableRow{
			Cells: []any{
				secret.Name,
				secret.Annotations["kargo.akuity.io/description"],
				duration.HumanDuration(time.Since(secret.CreationTimestamp.Time)),
			},
			Object: list.Items[i],
		}
	}
	return &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string"},
			{Name: "Description", Type: "string"},
			{Name: "Age", Type: "string"},
		},
		Rows: rows,
	}
}
