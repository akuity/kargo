package refresh

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/cli/config"
	"github.com/akuity/kargo/pkg/cli/option"
	"github.com/akuity/kargo/pkg/cli/templates"
	"github.com/akuity/kargo/pkg/client/watch"
	"github.com/akuity/kargo/pkg/server"
)

func newRefreshClusterConfigCommand(cfg config.CLIConfig) *cobra.Command {
	cmdOpts := &refreshOptions{
		Config:       cfg,
		ResourceType: server.RefreshResourceTypeClusterConfig,
	}

	cmd := &cobra.Command{
		Use:  "clusterconfig [--wait]",
		Args: option.NoArgs,
		Example: templates.Example(`
# Refresh the cluster configuration
kargo refresh clusterconfig

# Refresh the cluster configuration and wait for it to complete
kargo refresh clusterconfig --wait
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmdOpts.complete(nil)

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

func waitForClusterConfig(
	ctx context.Context,
	watchClient *watch.Client,
) error {
	eventCh, errCh := watchClient.WatchClusterConfig(ctx)
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				select {
				case err := <-errCh:
					if err != nil {
						return fmt.Errorf("watch clusterconfig: %w", err)
					}
				default:
				}
				return errors.New("unexpected end of watch stream")
			}
			if event.Object == nil {
				return errors.New("unexpected response")
			}
			token, ok := api.RefreshAnnotationValue(event.Object.GetAnnotations())
			if !ok {
				return fmt.Errorf(
					"ClusterConfig %q has no %q annotation",
					event.Object.Name, kargoapi.AnnotationKeyRefresh,
				)
			}
			if event.Object.Status.LastHandledRefresh == token {
				return nil
			}
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("watch clusterconfig: %w", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
