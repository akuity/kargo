package refresh

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/config"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

func newRefreshWarehouseCommand(cfg config.CLIConfig, opt *option.Option) *cobra.Command {
	cmdOpts := &refreshOptions{
		Option: opt,
		Config: cfg,
	}

	cmd := &cobra.Command{
		Use:     "warehouse (WAREHOUSE)",
		Args:    option.ExactArgs(1),
		Example: "kargo warehouse refresh --project=guestbook (WAREHOUSE)",
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
		return errors.Wrap(err, "watch warehouse")
	}
	defer func() {
		if conn, connErr := res.Conn(); connErr == nil {
			_ = conn.CloseRequest()
		}
	}()
	for {
		if !res.Receive() {
			if err = res.Err(); err != nil {
				return errors.Wrap(err, "watch warehouse")
			}
			return errors.New("unexpected end of watch stream")
		}
		msg := res.Msg()
		if msg == nil || msg.Warehouse == nil {
			return errors.New("unexpected response")
		}
		if msg.Warehouse.Metadata.Annotations == nil ||
			msg.Warehouse.Metadata.Annotations[kargoapi.AnnotationKeyRefresh] == "" {
			return nil
		}
	}
}
