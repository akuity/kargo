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

func newRefreshWarehouseCommand(cfg config.CLIConfig) *cobra.Command {
	cmdOpts := &refreshOptions{
		Config:       cfg,
		ResourceType: server.RefreshResourceTypeWarehouse,
	}

	cmd := &cobra.Command{
		Use:  "warehouse [--project=project] NAME [--wait]",
		Args: option.ExactArgs(1),
		Example: templates.Example(`
# Refresh a warehouse
kargo refresh warehouse --project=my-project my-warehouse

# Refresh a warehouse and wait for it to complete
kargo refresh warehouse --project=my-project my-warehouse --wait

# Refresh a warehouse in the default project
kargo config set-project my-project
kargo refresh warehouse my-warehouse
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

	return cmd
}

func waitForWarehouse(
	ctx context.Context,
	watchClient *watch.Client,
	project string,
	name string,
) error {
	eventCh, errCh := watchClient.WatchWarehouse(ctx, project, name)
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				select {
				case err := <-errCh:
					if err != nil {
						return fmt.Errorf("watch warehouse: %w", err)
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
				return fmt.Errorf( // nolint: staticcheck
					"Warehouse %q in Project %q has no %q annotation",
					name, project, kargoapi.AnnotationKeyRefresh,
				)
			}
			if event.Object.Status.LastHandledRefresh == token {
				return nil
			}
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("watch warehouse: %w", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
