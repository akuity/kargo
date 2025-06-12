package refresh

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	v1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/api/service/v1alpha1/svcv1alpha1connect"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
)

func newRefreshProjectConfigCommand(cfg config.CLIConfig) *cobra.Command {
	cmdOpts := &refreshOptions{
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use:  "projectconfig [--wait]",
		Args: option.NoArgs,
		Example: templates.Example(`
# Refresh the project configuration
kargo refresh projectconfig --project=my-project

# Refresh the project configuration and wait for it to complete
kargo refresh projectconfig --project=my-project --wait

# Refresh the project configuration in the default project
kargo config set-project my-project
kargo refresh projectconfig
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmdOpts.complete(refreshResourceTypeProjectConfig, nil)

			if err := cmdOpts.validate(true, false); err != nil {
				return err
			}

			return cmdOpts.run(cmd.Context())
		},
	}

	// Register the option flags on the command.
	cmdOpts.addFlags(cmd, true)

	return cmd
}

func waitForProjectConfig(
	ctx context.Context,
	kargoSvcCli svcv1alpha1connect.KargoServiceClient,
	project string,
) error {
	res, err := kargoSvcCli.WatchProjectConfig(ctx, connect.NewRequest(&v1alpha1.WatchProjectConfigRequest{
		Project: project,
	}))
	if err != nil {
		return fmt.Errorf("watch projectconfig: %w", err)
	}
	defer func() {
		if conn, connErr := res.Conn(); connErr == nil {
			_ = conn.CloseRequest()
		}
	}()
	for {
		if !res.Receive() {
			if err = res.Err(); err != nil {
				return fmt.Errorf("watch projectconfig: %w", err)
			}
			return errors.New("unexpected end of watch stream")
		}
		msg := res.Msg()
		if msg == nil || msg.ProjectConfig == nil {
			return errors.New("unexpected response")
		}
		token, ok := api.RefreshAnnotationValue(msg.ProjectConfig.GetAnnotations())
		if !ok {
			return fmt.Errorf(
				"ProjectConfig %q has no %q annotation",
				msg.ProjectConfig.Name, kargoapi.AnnotationKeyRefresh,
			)
		}
		if msg.ProjectConfig.Status.LastHandledRefresh == token {
			return nil
		}
	}
}
