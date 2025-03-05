package refresh

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	"github.com/akuity/kargo/internal/cli/templates"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

func newRefreshStageCommand(cfg config.CLIConfig) *cobra.Command {
	cmdOpts := &refreshOptions{
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use:  "stage [--project=project] NAME [--wait]",
		Args: option.ExactArgs(1),
		Example: templates.Example(`
# Refresh a stage
kargo refresh stage --project=my-project my-stage

# Refresh a stage and wait for it to complete
kargo refresh stage --project=my-project my-stage --wait

# Refresh a stage in the default project
kargo config set-project my-project
kargo refresh stage my-stage
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdOpts.complete(refreshResourceTypeStage, args)

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

func waitForStage(
	ctx context.Context,
	kargoSvcCli svcv1alpha1connect.KargoServiceClient,
	project string,
	name string,
) error {
	res, err := kargoSvcCli.WatchStages(ctx, connect.NewRequest(&v1alpha1.WatchStagesRequest{
		Project: project,
		Name:    name,
	}))
	if err != nil {
		return fmt.Errorf("watch stage: %w", err)
	}
	defer func() {
		if conn, connErr := res.Conn(); connErr == nil {
			_ = conn.CloseRequest()
		}
	}()
	for {
		if !res.Receive() {
			if err = res.Err(); err != nil {
				return fmt.Errorf("watch stage: %w", err)
			}
			return errors.New("unexpected end of watch stream")
		}
		msg := res.Msg()
		if msg == nil || msg.Stage == nil {
			return errors.New("unexpected response")
		}
		token, ok := api.RefreshAnnotationValue(msg.Stage.GetAnnotations())
		if !ok {
			return fmt.Errorf(
				"Stage %q in Project %q has no %q annotation",
				name, project, kargoapi.AnnotationKeyRefresh,
			)
		}
		if msg.Stage.Status.LastHandledRefresh == token {
			return nil
		}
	}
}
