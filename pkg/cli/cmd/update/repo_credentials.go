package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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

type updateRepoCredentialsOptions struct {
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

func newUpdateRepoCredentialsCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &updateRepoCredentialsOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use: `repo-credentials [--project=project | --shared] NAME \
    [--git | --helm | --image] \
    [--description=description] \
    [--repo-url=repo-url [--regex]] \
    [--username=username] \
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
		Short: "Update credentials for accessing a repository",
		Args:  cobra.ExactArgs(1),
		Example: templates.Example(`
# Update the password in my-credentials
kargo update repo-credentials --project=my-project my-credentials --password=my-password

# Update the username in my-credentials
kargo update repo-credentials --project=my-project my-credentials --username=my-username

# Update the credential type of my-credentials
kargo update repo-credentials --project=my-project my-credentials --git

# Update the password in my-credentials in the default project
kargo config set-project my-project
kargo update repo-credentials my-credentials --password=my-password

# Update the username in my-credentials in the default project
kargo config set-project my-project
kargo update repo-credentials my-credentials --username=my-username

# Update the credentials type of my-credentials in the default project
kargo config set-project my-project
kargo update repo-credentials my-credentials --git
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

// addFlags adds the flags for the update repo-credentials options to the provided
// command.
func (o *updateRepoCredentialsOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project in which to update credentials. If not set, the default project will be used.",
	)
	option.Shared(
		cmd.Flags(), &o.Shared, false,
		"Whether to update shared credentials instead of project-specific credentials.",
	)
	// project and shared flags are mutually exclusive
	cmd.MarkFlagsMutuallyExclusive(option.ProjectFlag, option.SharedFlag)

	option.Description(cmd.Flags(), &o.Description, "Change the description of the credentials.")
	option.Git(cmd.Flags(), &o.Git, "Change the credentials to be for a Git repository.")
	option.Helm(cmd.Flags(), &o.Helm, "Change the credentials to be for a Helm chart repository.")
	option.Image(cmd.Flags(), &o.Image, "Change the credentials to be for a container image repository.")
	option.Type(cmd.Flags(), &o.Type, "Type of repository the credentials are for.")
	option.RepoURL(cmd.Flags(), &o.RepoURL, "URL of the repository the credentials are for.")
	option.Regex(
		cmd.Flags(), &o.Regex,
		fmt.Sprintf(
			"Indicates that the value of --%s is a regular expression.",
			option.RepoURLFlag,
		),
	)
	option.Username(cmd.Flags(), &o.Username, "Change the username in the credentials.")
	option.Password(cmd.Flags(), &o.Password, "Change the password in the credentials.")

	cmd.MarkFlagsMutuallyExclusive(option.GitFlag, option.HelmFlag, option.ImageFlag, option.TypeFlag)
}

// complete sets the options from the command arguments.
func (o *updateRepoCredentialsOptions) complete(args []string) {
	o.Name = args[0]
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *updateRepoCredentialsOptions) validate() error {
	var errs []error
	if o.Project == "" && !o.Shared {
		errs = append(errs, fmt.Errorf(
			"either %s or %s is required", option.ProjectFlag, option.SharedFlag,
		))
	}

	if o.Regex && o.RepoURL == "" {
		errs = append(errs, errors.New("regex is only allowed when repo-url is set"))
	}

	// At least one update field must be provided
	hasUpdate := o.Description != "" ||
		o.Git || o.Helm || o.Image || o.Type != "" ||
		o.RepoURL != "" ||
		o.Username != "" ||
		o.Password != ""
	if !hasUpdate {
		errs = append(errs, errors.New(
			"at least one of --description, --git, --helm, --image, --type, "+
				"--repo-url, --username, or --password must be provided",
		))
	}

	return errors.Join(errs...)
}

// run updates the credentials based on the options.
func (o *updateRepoCredentialsOptions) run(ctx context.Context) error {
	apiClient, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	// Resolve the type flag
	credType := o.Type
	if o.Git {
		credType = credentials.TypeGit.String()
	} else if o.Helm {
		credType = credentials.TypeHelm.String()
	} else if o.Image {
		credType = credentials.TypeImage.String()
	}

	// Build the patch request with only fields that need to change
	patchReq := &models.PatchRepoCredentialsRequest{
		Description:    o.Description,
		Type:           credType,
		RepoURL:        o.RepoURL,
		RepoURLIsRegex: o.Regex,
		Username:       o.Username,
		Password:       o.Password,
	}

	var payload any

	switch {
	case o.Shared:
		_, err = apiClient.Credentials.PatchSharedRepoCredentials(
			credclient.NewPatchSharedRepoCredentialsParams().
				WithRepoCredentials(o.Name).
				WithBody(patchReq),
			nil,
		)
		if err != nil {
			return fmt.Errorf("patch shared repo credentials: %w", err)
		}

		// Get the updated credentials
		var res *credclient.GetSharedRepoCredentialsOK
		if res, err = apiClient.Credentials.GetSharedRepoCredentials(
			credclient.NewGetSharedRepoCredentialsParams().WithRepoCredentials(o.Name),
			nil,
		); err != nil {
			return fmt.Errorf("get shared repo credentials: %w", err)
		}
		payload = res.GetPayload()

	default:
		_, err = apiClient.Credentials.PatchProjectRepoCredentials(
			credclient.NewPatchProjectRepoCredentialsParams().
				WithProject(o.Project).
				WithRepoCredentials(o.Name).
				WithBody(patchReq),
			nil,
		)
		if err != nil {
			return fmt.Errorf("patch project repo credentials: %w", err)
		}

		// Get the updated credentials
		res, err := apiClient.Credentials.GetProjectRepoCredentials(
			credclient.NewGetProjectRepoCredentialsParams().
				WithProject(o.Project).
				WithRepoCredentials(o.Name),
			nil,
		)
		if err != nil {
			return fmt.Errorf("get project repo credentials: %w", err)
		}
		payload = res.GetPayload()
	}

	return o.printCredentials(payload)
}

func (o *updateRepoCredentialsOptions) printCredentials(payload any) error {
	credJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	var cred *corev1.Secret
	if err = json.Unmarshal(credJSON, &cred); err != nil {
		return fmt.Errorf("unmarshal credentials: %w", err)
	}

	printer, err := o.ToPrinter()
	if err != nil {
		return fmt.Errorf("new printer: %w", err)
	}
	return printer.PrintObj(cred, o.Out)
}
