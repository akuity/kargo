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

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/generated/credentials"
	libCreds "github.com/akuity/kargo/pkg/credentials"
)

type getRepoCredentialsOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	*getOptions

	Config        config.CLIConfig
	ClientOptions client.Options

	Shared  bool
	Project string
	Names   []string
}

func newGetRepoCredentialsCommand(
	cfg config.CLIConfig,
	streams genericiooptions.IOStreams,
	getOptions *getOptions,
) *cobra.Command {
	cmdOpts := &getRepoCredentialsOptions{
		Config:     cfg,
		IOStreams:  streams,
		getOptions: getOptions,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use: "repo-credentials [--project=project] [NAME ...] [--no-headers]",
		Aliases: []string{
			"repo-credential",
			"repo-creds",
			"repo-cred",
			"repocredentials",
			"repocredential",
			"repocreds",
			"repocred",
		},
		Short: "Display one or many repository credentials",
		Example: templates.Example(`
# List all repository credentials in my-project
kargo get repo-credentials --project=my-project

# Get specific repository credentials in my-project
kargo get repo-credentials --project=my-project my-credentials

# List all repository credentials in the default project
kargo config set-project my-project
kargo get repo-credentials

# Get specific repository credentials in the default project
kargo config set-project my-project
kargo get repo-credentials my-credentials

# List shared repository credentials
kargo get repo-credentials --shared

# Get specific shared repository credentials
kargo get repo-credentials --shared my-credentials
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

// addFlags adds the flags for the get repo-credentials options to the provided
// command.
func (o *getRepoCredentialsOptions) addFlags(cmd *cobra.Command) {
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
	// project and shared flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SharedFlag)
}

// complete sets the options from the command arguments.
func (o *getRepoCredentialsOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *getRepoCredentialsOptions) validate() error {
	var errs []error
	if o.Project == "" && !o.Shared {
		errs = append(errs, fmt.Errorf(
			"either %s or %s is required", option.ProjectFlag, option.SharedFlag,
		))
	}
	return errors.Join(errs...)
}

// run gets the credentials from the server and prints them to the console.
func (o *getRepoCredentialsOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	if len(o.Names) == 0 {
		var payload any
		switch {
		case o.Shared:
			var res *credentials.ListSharedRepoCredentialsOK
			if res, err = apiClient.Credentials.ListSharedRepoCredentials(
				credentials.NewListSharedRepoCredentialsParams(),
				nil,
			); err != nil {
				return fmt.Errorf("list shared credentials: %w", err)
			}
			payload = res.Payload
		default:
			var res *credentials.ListProjectRepoCredentialsOK
			if res, err = apiClient.Credentials.ListProjectRepoCredentials(
				credentials.NewListProjectRepoCredentialsParams().WithProject(o.Project),
				nil,
			); err != nil {
				return fmt.Errorf("list project credentials: %w", err)
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
		return PrintObjects(creds.Items, o.PrintFlags, o.IOStreams, o.NoHeaders)
	}

	res := make([]*corev1.Secret, 0, len(o.Names))
	errs := make([]error, 0, len(o.Names))
	for _, name := range o.Names {
		var payload any
		switch {
		case o.Shared:
			var res *credentials.GetSharedRepoCredentialsOK
			if res, err = apiClient.Credentials.GetSharedRepoCredentials(
				credentials.NewGetSharedRepoCredentialsParams().
					WithRepoCredentials(name),
				nil,
			); err != nil {
				errs = append(errs, err)
				continue
			}
			payload = res.Payload
		default:
			var res *credentials.GetProjectRepoCredentialsOK
			if res, err = apiClient.Credentials.GetProjectRepoCredentials(
				credentials.NewGetProjectRepoCredentialsParams().
					WithProject(o.Project).
					WithRepoCredentials(name),
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
		res = append(res, cred)
	}

	if err = PrintObjects(res, o.PrintFlags, o.IOStreams, o.NoHeaders); err != nil {
		return fmt.Errorf("print credentials: %w", err)
	}
	return errors.Join(errs...)
}

func newRepoCredentialsTable(list *metav1.List) *metav1.Table {
	rows := make([]metav1.TableRow, len(list.Items))
	for i, item := range list.Items {
		secret := item.Object.(*corev1.Secret) // nolint: forcetypeassert
		rows[i] = metav1.TableRow{
			Cells: []any{
				secret.Name,
				secret.Labels[kargoapi.LabelKeyCredentialType],
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
