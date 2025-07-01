package update

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/genericiooptions"

	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/io"
	"github.com/akuity/kargo/internal/cli/kubernetes"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
	"github.com/akuity/kargo/internal/credentials"
)

type updateCredentialsOptions struct {
	genericiooptions.IOStreams
	*genericclioptions.PrintFlags

	Config        config.CLIConfig
	ClientOptions client.Options

	Project                     string
	Name                        string
	Description                 string
	Git                         bool
	Helm                        bool
	Image                       bool
	Type                        string
	RepoURL                     string
	Regex                       bool
	Username                    string
	Password                    string
	ChangePasswordInteractively bool
}

func newUpdateCredentialsCommand(cfg config.CLIConfig, streams genericiooptions.IOStreams) *cobra.Command {
	cmdOpts := &updateCredentialsOptions{
		Config:     cfg,
		IOStreams:  streams,
		PrintFlags: genericclioptions.NewPrintFlags("").WithTypeSetter(kubernetes.GetScheme()),
	}

	cmd := &cobra.Command{
		Use: `credentials [--project=project] NAME \
    [--git | --helm | --image] \
    [--description=description] \
    [--repo-url=repo-url [--regex]] \
    [--username=username] \
    [--password=password | --interactive-password]`,
		Aliases: []string{"credential", "creds", "cred"},
		Short:   "Update credentials for accessing a repository",
		Args:    cobra.ExactArgs(1),
		Example: templates.Example(`
# Update the password in my-credentials
kargo update credentials --project=my-project my-credentials --password=my-password

# Update the username in my-credentials
kargo update credentials --project=my-project my-credentials --username=my-username

# Update the credential type of my-credentials
kargo update credentials --project=my-project my-credentials --git

# Update the password in my-credentials in the default project
kargo config set-project my-project
kargo update credentials my-credentials --password=my-password

# Update the username in my-credentials in the default project
kargo config set-project my-project
kargo update credentials my-credentials --username=my-username

# Update the credentials type of my-credentials in the default project
kargo config set-project my-project
kargo update credentials my-credentials --git
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

// addFlags adds the flags for the get credentials options to the provided
// command.
func (o *updateCredentialsOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())
	o.PrintFlags.AddFlags(cmd)
	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project in which to update credentials. If not set, the default project will be used.",
	)
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
	option.InteractivePassword(
		cmd.Flags(),
		&o.ChangePasswordInteractively,
		"Change the password in the credentials interactively.",
	)

	cmd.MarkFlagsMutuallyExclusive(option.GitFlag, option.HelmFlag, option.ImageFlag, option.TypeFlag)

	cmd.MarkFlagsMutuallyExclusive(option.PasswordFlag, option.InteractivePasswordFlag)
}

// complete sets the options from the command arguments.
func (o *updateCredentialsOptions) complete(args []string) {
	o.Name = args[0]
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *updateCredentialsOptions) validate() error {
	// While the flags are marked as required, a user could still provide an empty
	// string. This is a check to ensure that the flags are not empty.
	if o.Project == "" {
		return errors.New("project is required")
	}

	if o.Regex && o.RepoURL == "" {
		return errors.New("regex is only allows when repo-url is set")
	}

	return nil
}

// run creates the credentials in the project based on the options.
func (o *updateCredentialsOptions) run(ctx context.Context) error {
	if o.ChangePasswordInteractively {
		for {
			if o.Password != "" {
				break
			}
			prompt := &survey.Password{
				Message: "Repository password",
			}
			if err := survey.AskOne(prompt, &o.Password); err != nil {
				return err
			}
		}
	}

	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
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

	resp, err := kargoSvcCli.UpdateCredentials(
		ctx,
		connect.NewRequest(
			&v1alpha1.UpdateCredentialsRequest{
				Project:        o.Project,
				Name:           o.Name,
				Description:    o.Description,
				Type:           o.Type,
				RepoUrl:        o.RepoURL,
				RepoUrlIsRegex: o.Regex,
				Username:       o.Username,
				Password:       o.Password,
			},
		),
	)
	if err != nil {
		return fmt.Errorf("update credentials: %w", err)
	}

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return fmt.Errorf("new printer: %w", err)
	}
	return printer.PrintObj(resp.Msg.GetCredentials(), o.IOStreams.Out)
}
