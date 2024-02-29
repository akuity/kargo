package delete

import (
	"context"
	goerrors "errors"
	"fmt"
	"slices"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type deleteCredentialsOptions struct {
	*option.Option
	Config config.CLIConfig

	Names []string
}

func newDeleteCredentialsCommand(
	cfg config.CLIConfig,
	opt *option.Option,
) *cobra.Command {
	cmdOpts := &deleteCredentialsOptions{
		Option: opt,
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use:     "credentials [--project=project] [NAME ...]",
		Aliases: []string{"credential", "creds", "cred"},
		Short:   "Delete credentials by name",
		Args:    cobra.MinimumNArgs(1),
		Example: `
# Delete credentials
kargo delete credentials --project=my-project my-credentials

# Delete multiple credentials
kargo delete credentials --project=my-project my-credentials1 my-credentials2

# Delete credentials from default project
kargo config set-project my-project
kargo delete credentials my-credentials
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
func (o *deleteCredentialsOptions) addFlags(cmd *cobra.Command) {
	o.PrintFlags.AddFlags(cmd)

	option.Project(
		cmd.Flags(), &o.Project, o.Project,
		"The project for which to delete credentials. If not set, the default project will be used.",
	)
}

// complete sets the options from the command arguments.
func (o *deleteCredentialsOptions) complete(args []string) {
	o.Names = slices.Compact(args)
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *deleteCredentialsOptions) validate() error {
	var errs []error

	if o.Project == "" {
		errs = append(errs, errors.New("project is required"))
	}

	if len(o.Names) == 0 {
		errs = append(errs, errors.New("name is required"))
	}

	return goerrors.Join(errs...)
}

// run removes the credentials from the project based on the options.
func (o *deleteCredentialsOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.Option)
	if err != nil {
		return errors.Wrap(err, "get client from config")
	}

	var resErr error
	for _, name := range o.Names {
		if _, err := kargoSvcCli.DeleteCredentials(
			ctx,
			connect.NewRequest(
				&v1alpha1.DeleteCredentialsRequest{
					Project: o.Project,
					Name:    name,
				},
			),
		); err != nil {
			resErr = goerrors.Join(resErr, errors.Wrap(err, "Error"))
			continue
		}
		_, _ = fmt.Fprintf(o.IOStreams.Out, "Credentials Deleted: %q\n", name)
	}
	return resErr
}
