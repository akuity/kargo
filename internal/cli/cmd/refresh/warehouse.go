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

func newRefreshWarehouseCommand(cfg config.CLIConfig) *cobra.Command {
	cmdOpts := &refreshOptions{
		Config: cfg,
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
			cmdOpts.complete(refreshResourceTypeWarehouse, args)

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
	kargoSvcCli svcv1alpha1connect.KargoServiceClient,
	project string,
	name string,
) error {
	res, err := kargoSvcCli.WatchWarehouses(ctx, connect.NewRequest(&v1alpha1.WatchWarehousesRequest{
		Project: project,
		Name:    name,
	}))
	if err != nil {
		return fmt.Errorf("watch warehouse: %w", err)
	}
	defer func() {
		if conn, connErr := res.Conn(); connErr == nil {
			_ = conn.CloseRequest()
		}
	}()
	for {
		if !res.Receive() {
			if err = res.Err(); err != nil {
				return fmt.Errorf("watch warehouse: %w", err)
			}
			return errors.New("unexpected end of watch stream")
		}
		msg := res.Msg()
		if msg == nil || msg.Warehouse == nil {
			return errors.New("unexpected response")
		}
		token, ok := api.RefreshAnnotationValue(msg.Warehouse.GetAnnotations())
		if !ok {
			return fmt.Errorf(
				"Warehouse %q in Project %q has no %q annotation",
				name, project, kargoapi.AnnotationKeyRefresh,
			)
		}
		if msg.Warehouse.Status.LastHandledRefresh == token {
			return nil
		}
	}
}
