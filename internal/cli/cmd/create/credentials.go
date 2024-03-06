package create

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
	"github.com/akuity/kargo/internal/credentials"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type createCredentialsOptions struct {
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

func newCreateCredentialsCommand(
	cfg config.CLIConfig,
	opt *option.Option,
) *cobra.Command {
	cmdOpts := &createCredentialsOptions{
		Option: opt,
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use: `credentials [--project=project] NAME \
    (--git | --helm | --image) \
    --repo-url=repo-url | --repo-url-pattern=repo-url-pattern) \
    -username=username \
    [--password=password]`,
		Aliases: []string{"credential", "creds", "cred"},
		Short:   "Create new credentials for accessing a repository",
		Args:    cobra.ExactArgs(1),
		Example: `
# Create credentials for a Git repository
kargo create credentials --project=my-project my-credentials \
  --git --repo-url=https://github.com/my-org/my-repo.git \
  --username=my-username --password=my-password

# Create credentials for a Helm chart repository
kargo create credentials --project=my-project my-credentials \
  --helm --repo-url=oci://ghcr.io/my-org/my-chart \
  --username=my-username --password=my-password

# Create credentials for a container image repository
kargo create credentials --project=my-project my-credentials \
  --image --repo-url=ghcr.io/my-org/my-image \
  --username=my-username --password=my-password

# Create credentials for a Git repository in the default project
kargo config set-project my-project
kargo create credentials my-credentials \
  --git --repo-url=https://github.com/my-org/my-repo.git \
  --username=my-username --password=my-password

# Create credentials for a Helm chart repository in the default project
kargo config set-project my-project
kargo create credentials my-credentials \
  --helm --repo-url=oci://ghcr.io/my-org/my-chart \
  --username=my-username --password=my-password

# Create credentials for a container image repository in the default project
kargo config set-project my-project
kargo create credentials my-credentials \
  --image --repo-url=ghcr.io/my-org/my-image \
  --username=my-username --password=my-password`,
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
func (o *createCredentialsOptions) addFlags(cmd *cobra.Command) {
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
func (o *createCredentialsOptions) complete(args []string) {
	o.Name = args[0]
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *createCredentialsOptions) validate() error {
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
func (o *createCredentialsOptions) run(ctx context.Context) error {
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
		repoType = string(credentials.TypeGit)
	} else if o.Helm {
		repoType = string(credentials.TypeHelm)
	} else if o.Image {
		repoType = string(credentials.TypeImage)
	}

	if _, err := kargoSvcCli.CreateCredentials(
		ctx,
		connect.NewRequest(
			&v1alpha1.CreateCredentialsRequest{
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
		return errors.Wrap(err, "create credentials")
	}

	return nil
}
