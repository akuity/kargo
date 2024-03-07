package update

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/AlecAivazis/survey/v2"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/utils/ptr"

	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/credentials"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type updateCredentialsOptions struct {
	*option.Option
	Config config.CLIConfig

	Name                        string
	Git                         bool
	Helm                        bool
	Image                       bool
	Type                        string
	RepoURL                     string
	RepoURLPattern              string
	Username                    string
	Password                    string
	ChangePasswordInteractively bool
}

func newUpdateCredentialsCommand(
	cfg config.CLIConfig,
	opt *option.Option,
) *cobra.Command {
	cmdOpts := &updateCredentialsOptions{
		Option: opt,
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use: `credentials [--project=project] NAME \
    [--git | --helm | --image] \
    [--repo-url=repo-url | --repo-url-pattern=repo-url-pattern] \
    [--username=username] \
    [--password=password | --interactive-password]`,
		Aliases: []string{"credential", "creds", "cred"},
		Short:   "Update credentials for accessing a repository",
		Args:    cobra.ExactArgs(1),
		Example: `
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
kargo update credentials my-credentials --git`,
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

	return cmd

}

// addFlags adds the flags for the get credentials options to the provided
// command.
func (o *updateCredentialsOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)
	option.Project(
		cmd.Flags(), &o.Project, o.Project,
		"The project in which to update credentials. If not set, the default project will be used.",
	)
	option.Git(cmd.Flags(), &o.Git, "Change the credentials to be for a Git repository.")
	option.Helm(cmd.Flags(), &o.Helm, "Change the credentials to be for a Helm chart repository.")
	option.Image(cmd.Flags(), &o.Image, "Change the credentials to be for a container image repository.")
	option.Type(cmd.Flags(), &o.Type, "Type of repository the credentials are for.")
	option.RepoURL(cmd.Flags(), &o.RepoURL, "URL of the repository the credentials are for.")
	option.RepoURLPattern(
		cmd.Flags(), &o.RepoURLPattern,
		"Regular expression matching multiple repositories the credentials are for.",
	)
	option.Username(cmd.Flags(), &o.Username, "Change the username in the credentials.")
	option.Password(cmd.Flags(), &o.Password, "Change the password in the credentials.")
	option.InteractivePassword(
		cmd.Flags(),
		&o.ChangePasswordInteractively,
		"Change the password in the credentials interactively.",
	)

	cmd.MarkFlagsMutuallyExclusive(option.GitFlag, option.HelmFlag, option.ImageFlag, option.TypeFlag)

	cmd.MarkFlagsMutuallyExclusive(option.RepoURLFlag, option.RepoURLPatternFlag)

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

	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.Option)
	if err != nil {
		return errors.Wrap(err, "get client from config")
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
				Type:           o.Type,
				RepoUrl:        o.RepoURL,
				RepoUrlPattern: o.RepoURLPattern,
				Username:       o.Username,
				Password:       o.Password,
			},
		),
	)
	if err != nil {
		return errors.Wrap(err, "update credentials")
	}

	if ptr.Deref(o.PrintFlags.OutputFormat, "") == "" {
		_, _ = fmt.Fprintf(o.IOStreams.Out, "Credentials Updated: %q\n", o.Name)
		return nil
	}

	secret := typesv1alpha1.FromSecretProto(resp.Msg.GetCredentials())

	printer, err := o.PrintFlags.ToPrinter()
	if err != nil {
		return errors.Wrap(err, "new printer")
	}
	return printer.PrintObj(secret, o.IOStreams.Out)
}
