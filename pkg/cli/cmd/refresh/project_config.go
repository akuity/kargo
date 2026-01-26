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

func newRefreshProjectConfigCommand(cfg config.CLIConfig) *cobra.Command {
	cmdOpts := &refreshOptions{
		Config:       cfg,
		ResourceType: server.RefreshResourceTypeProjectConfig,
	}

	cmd := &cobra.Command{
		Use:  "projectconfig [--project=project] [--wait]",
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

func waitForProjectConfig(
	ctx context.Context,
	watchClient *watch.Client,
	project string,
) error {
	eventCh, errCh := watchClient.WatchProjectConfig(ctx, project)
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				select {
				case err := <-errCh:
					if err != nil {
						return fmt.Errorf("watch projectconfig: %w", err)
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
					"ProjectConfig %q has no %q annotation",
					event.Object.Name, kargoapi.AnnotationKeyRefresh,
				)
			}
			if event.Object.Status.LastHandledRefresh == token {
				return nil
			}
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("watch projectconfig: %w", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
