package create

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	"github.com/akuity/kargo/pkg/cli/client"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/io"
	"github.com/akuity/kargo/pkg/cli/kubernetes"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	credclient "github.com/akuity/kargo/pkg/client/generated/credentials"
	"github.com/akuity/kargo/pkg/client/generated/models"
	"github.com/akuity/kargo/pkg/credentials"
)

type createRepoCredentialsOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Shared      bool
	Project     string
	Name        string
	Description string
	Git         bool
	Helm        bool
	Image       bool
	Type        string
	RepoURL     string
	Regex       bool
	Username    string
	Password    string
}

func newRepoCredentialsCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &createRepoCredentialsOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("created").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use: `repo-credentials [--project=project] NAME \
    (--git | --helm | --image) \
    [--description=description] \
    --repo-url=repo-url [--regex] \
    -username=username \
    [--password=password]`,
		Aliases: []string{
			"repo-credential",
			"repo-creds",
			"repo-cred",
			"repocredentials",
			"repocredential",
			"repocreds",
			"repocred",
		},
		Short: "Create new credentials for accessing a repository",
		Args:  cobra.ExactArgs(1),
		Example: templates.Example(`
# Create credentials for a Git repository
kargo create repo-credentials --project=my-project my-credentials \
  --git --repo-url=https://github.com/my-org/my-repo.git \
  --username=my-username --password=my-password

# Create credentials for a Helm chart repository
kargo create repo-credentials --project=my-project my-credentials \
  --helm --repo-url=oci://ghcr.io/my-org/my-chart \
  --username=my-username --password=my-password

# Create credentials for a container image repository
kargo create repo-credentials --project=my-project my-credentials \
  --image --repo-url=ghcr.io/my-org/my-image \
  --username=my-username --password=my-password

# Create credentials for a Git repository in the default project
kargo config set-project my-project
kargo create repo-credentials my-credentials \
  --git --repo-url=https://github.com/my-org/my-repo.git \
  --username=my-username --password=my-password

# Create credentials for a Helm chart repository in the default project
kargo config set-project my-project
kargo create repo-credentials my-credentials \
  --helm --repo-url=oci://ghcr.io/my-org/my-chart \
  --username=my-username --password=my-password

# Create credentials for a container image repository in the default project
kargo config set-project my-project
kargo create repo-credentials my-credentials \
  --image --repo-url=ghcr.io/my-org/my-image \
  --username=my-username --password=my-password

# Create shared credentials for all GitHub repositories
kargo create repo-credentials --shared my-credentials \
	--git --repo-url='^https://github\.com/.*$' --regex \
	--username=my-username --password=personal-access-token
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

// addFlags adds the flags for the repo-credentials options to the provided
// command.
func (o *createRepoCredentialsOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project in which to create credentials. If not set, the default project will be used.",
	)
	option.Shared(
		cmd.Flags(), &o.Shared, false,
		"Whether to create shared credentials that can be used across all projects.",
	)
	// project and shared flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SharedFlag)

	option.Description(cmd.Flags(), &o.Description, "Description of the credentials.")
	option.Git(cmd.Flags(), &o.Git, "Create credentials for a Git repository.")
	option.Helm(cmd.Flags(), &o.Helm, "Create credentials for a Helm chart repository.")
	option.Image(cmd.Flags(), &o.Image, "Create credentials for a container image repository.")
	option.Type(cmd.Flags(), &o.Type, "Type of repository the credentials are for.")
	option.RepoURL(cmd.Flags(), &o.RepoURL, "URL of the repository the credentials are for.")
	option.Regex(
		cmd.Flags(), &o.Regex,
		fmt.Sprintf(
			"Indicates that the value of --%s is a regular expression.",
			option.RepoURLFlag,
		),
	)
	option.Username(cmd.Flags(), &o.Username, "Username for the credentials.")
	option.Password(cmd.Flags(), &o.Password, "Password for the credentials.")

	cmd.MarkFlagsOneRequired(option.GitFlag, option.HelmFlag, option.ImageFlag, option.TypeFlag)
	cmd.MarkFlagsMutuallyExclusive(option.GitFlag, option.HelmFlag, option.ImageFlag, option.TypeFlag)

	if err := cmd.MarkFlagRequired(option.RepoURLFlag); err != nil {
		panic(
			fmt.Errorf(
				"could not mark %s flag as required: %w",
				option.RepoURLFlag,
				err,
			),
		)
	}

	if err := cmd.MarkFlagRequired(option.UsernameFlag); err != nil {
		panic(
			fmt.Errorf(
				"could not mark %s flag as required: %w",
				option.UsernameFlag,
				err,
			),
		)
	}
}

// complete sets the options from the command arguments.
func (o *createRepoCredentialsOptions) complete(args []string) {
	o.Name = args[0]
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *createRepoCredentialsOptions) validate() error {
	var errs []error
	if o.Project == "" && !o.Shared {
		errs = append(errs, fmt.Errorf(
			"either %s or %s is required", option.ProjectFlag, option.SharedFlag,
		))
	}

	// While these flags are marked as required, a user could still provide an
	// empty string. This is a check to ensure that the flags are not empty.
	if o.RepoURL == "" {
		errs = append(errs, errors.New("repo-url is required"))
	}
	if o.Username == "" {
		errs = append(errs, errors.New("username is required"))
	}
	return errors.Join(errs...)
}

// run creates the credentials in the project based on the options.
func (o *createRepoCredentialsOptions) run(ctx context.Context) error {
	for o.Password == "" {

		prompt := &survey.Password{
			Message: "Repository password",
		}
		if err := survey.AskOne(prompt, &o.Password); err != nil {
			return err
		}
	}

	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	if o.Git {
		o.Type = credentials.TypeGit.String()
	} else if o.Helm {
		o.Type = credentials.TypeHelm.String()
	} else if o.Image {
		o.Type = credentials.TypeImage.String()
	}

	createReq := &models.CreateRepoCredentialsRequest{
		Description:    o.Description,
		Name:           o.Name,
		Password:       o.Password,
		RepoURL:        o.RepoURL,
		RepoURLIsRegex: o.Regex,
		Type:           o.Type,
		Username:       o.Username,
	}

	var payload any
	switch {
	case o.Shared:
		var res *credclient.CreateSharedRepoCredentialsCreated
		if res, err = apiClient.Credentials.CreateSharedRepoCredentials(
			credclient.NewCreateSharedRepoCredentialsParams().
				WithBody(createReq),
			nil,
		); err != nil {
			return fmt.Errorf("create shared credentials: %w", err)
		}
		payload = res.GetPayload()
	default:
		var res *credclient.CreateProjectRepoCredentialsCreated
		if res, err = apiClient.Credentials.CreateProjectRepoCredentials(
			credclient.NewCreateProjectRepoCredentialsParams().
				WithProject(o.Project).
				WithBody(createReq),
			nil,
		); err != nil {
			return fmt.Errorf("create project credentials: %w", err)
		}
		payload = res.GetPayload()
	}

	resJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}

	var secret *corev1.Secret
	if err = json.Unmarshal(resJSON, &secret); err != nil {
		return fmt.Errorf("unmarshal secret: %w", err)
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("new printer: %w", err)
	}
	return printer.PrintObj(secret, o.Out)
}
