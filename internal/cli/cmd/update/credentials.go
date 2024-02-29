package update

import (
	"context"
	goerrors "errors"

	"connectrpc.com/connect"
	"github.com/AlecAivazis/survey/v2"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type updateCredentialsOptions struct {
	*option.Option
	Config config.CLIConfig

	Name           string
	Git            bool
	Helm           bool
	Image          bool
	RepoURL        string
	RepoURLPattern string
	Username       string
	Password       string
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
		Use: "credentials [--project=project] NAME (--git | --helm | --image) " +
			"(--repo-url=repo-url | --repo-url-pattern=repo-url-pattern) --username=username [--password=password]",
		Aliases: []string{"credential", "creds", "cred"},
		Short:   "Update credentials for accessing a repository",
		Args:    cobra.ExactArgs(1),
		Example: `
# Update my-credentials for a Git repository
kargo update credentials --project=my-project my-credentials \
  --git --repo-url=https://github.com/my-org/my-repo.git \
  --username=my-username --password=my-password

# Update my-credentials for a Helm chart repository
kargo update credentials --project=my-project my-credentials \
  --helm --repo-url=oci://ghcr.io/my-org/my-chart \
  --username=my-username --password=my-password

# Update my-credentials for a container image repository
kargo update credentials --project=my-project my-credentials \
  --image --repo-url=ghcr.io/my-org/my-image \
  --username=my-username --password=my-password

# Update my-credentials for a Git repository in the default project
kargo config set-project my-project
kargo update credentials my-credentials \
  --git --repo-url=https://github.com/my-org/my-repo.git \
  --username=my-username --password=my-password

# Update my-credentials for a Helm chart repository in the default project
kargo config set-project my-project
kargo update credentials my-credentials \
  --helm --repo-url=oci://ghcr.io/my-org/my-chart \
  --username=my-username --password=my-password

# Update credentials for a container image repository in the default project
kargo config set-project my-project
kargo update credentials my-credentials \
  --image --repo-url=ghcr.io/my-org/my-image \
  --username=my-username --password=my-password
`,
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
		"The project in which to create credentials. If not set, the default project will be used.",
	)
	option.Git(cmd.Flags(), &o.Git, "Create credentials for a Git repository.")
	option.Helm(cmd.Flags(), &o.Helm, "Create credentials for a Helm chart repository.")
	option.Image(cmd.Flags(), &o.Image, "Create credentials for a container image repository.")
	option.RepoURL(cmd.Flags(), &o.RepoURL, "URL of the repository the credentials are for.")
	option.RepoURLPattern(
		cmd.Flags(), &o.RepoURLPattern,
		"Regular expression matching multiple repositories the credentials are for.",
	)
	option.Username(cmd.Flags(), &o.Username, "Username for the credentials.")
	option.Password(cmd.Flags(), &o.Password, "Password for the credentials.")

	cmd.MarkFlagsOneRequired(option.GitFlag, option.HelmFlag, option.ImageFlag)
	cmd.MarkFlagsMutuallyExclusive(option.GitFlag, option.HelmFlag, option.ImageFlag)

	cmd.MarkFlagsOneRequired(option.RepoURLFlag, option.RepoURLPatternFlag)
	cmd.MarkFlagsMutuallyExclusive(option.RepoURLFlag, option.RepoURLPatternFlag)

	if err := cmd.MarkFlagRequired(option.UsernameFlag); err != nil {
		panic(errors.Wrapf(err, "could not mark %s flag as required", option.UsernameFlag))
	}
}

// complete sets the options from the command arguments.
func (o *updateCredentialsOptions) complete(args []string) {
	o.Name = args[0]
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *updateCredentialsOptions) validate() error {
	var errs []error
	// While the flags are marked as required, a user could still provide an empty
	// string. This is a check to ensure that the flags are not empty.
	if o.Project == "" {
		errs = append(errs, errors.New("project is required"))
	}
	if o.RepoURL == "" && o.RepoURLPattern == "" {
		errs = append(
			errs,
			errors.New("either repo-url or repo-url-pattern is required"),
		)
	}
	if o.Username == "" {
		errs = append(errs, errors.New("username is required"))
	}
	return goerrors.Join(errs...)
}

// run creates the credentials in the project based on the options.
func (o *updateCredentialsOptions) run(ctx context.Context) error {
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

	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.Option)
	if err != nil {
		return errors.Wrap(err, "get client from config")
	}

	var repoType string
	if o.Git {
		repoType = "git"
	} else if o.Helm {
		repoType = "helm"
	} else if o.Image {
		repoType = "image"
	}

	if _, err := kargoSvcCli.UpdateCredentials(
		ctx,
		connect.NewRequest(
			&v1alpha1.UpdateCredentialsRequest{
				Project:        o.Project,
				Name:           o.Name,
				Type:           repoType,
				RepoUrl:        o.RepoURL,
				RepoUrlPattern: o.RepoURLPattern,
				Username:       o.Username,
				Password:       o.Password,
			},
		),
	); err != nil {
		return errors.Wrap(err, "update credentials")
	}

	return nil
}
