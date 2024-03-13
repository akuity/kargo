package verify

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	"github.com/akuity/kargo/internal/cli/client"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type verifyStageOptions struct {
	Config        config.CLIConfig
	ClientOptions client.Options

	Project string
	Name    string
}

func newVerifyStageCommand(cfg config.CLIConfig) *cobra.Command {
	cmdOpts := &verifyStageOptions{
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use:   "stage [--project=project] (NAME)",
		Short: "Run the verification of a stage",
		Args:  option.ExactArgs(1),
		Example: `
# Run the verification of a stage
kargo verify stage --project=my-project my-stage

# Run the verification of a stage in the default project
kargo config set-project my-project
kargo verify stage my-stage
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

// addFlags adds the flags for the verify stage options to the provided command.
func (o *verifyStageOptions) addFlags(cmd *cobra.Command) {
	o.ClientOptions.AddFlags(cmd.PersistentFlags())

	option.Project(
		cmd.Flags(), &o.Project, o.Config.Project,
		"The project the stage belongs to. If not set, the default project will be used.",
	)
}

// complete sets the options from the command arguments.
func (o *verifyStageOptions) complete(args []string) {
	o.Name = strings.TrimSpace(strings.ToLower(args[0]))
}

// validate performs validation of the options. If the options are invalid, an
// error is returned.
func (o *verifyStageOptions) validate() error {
	var errs []error
	// While the flags are marked as required, a user could still provide an empty
	// string. This is a check to ensure that the flags are not empty.
	if o.Project == "" {
		errs = append(errs, fmt.Errorf("%s is required", option.ProjectFlag))
	}
	if o.Name == "" {
		errs = append(errs, errors.New("name is required"))
	}
	return errors.Join(errs...)
}

// run requests a rerun of the stage verification.
func (o *verifyStageOptions) run(ctx context.Context) error {
	kargoSvcCli, err := client.GetClientFromConfig(ctx, o.Config, o.ClientOptions)
	if err != nil {
		return fmt.Errorf("get client from config: %w", err)
	}

	if _, err := kargoSvcCli.RequestStageVerification(
		ctx,
		connect.NewRequest(
			&v1alpha1.RequestStageVerificationRequest{
				Project: o.Project,
				Name:    o.Name,
			},
		),
	); err != nil {
		return fmt.Errorf("verify stage: %w", err)
	}
	return nil
}
