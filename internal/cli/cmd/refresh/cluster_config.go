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

func newRefreshClusterConfigCommand(cfg config.CLIConfig) *cobra.Command {
	cmdOpts := &refreshOptions{
		Config:       cfg,
		ResourceType: refreshResourceTypeClusterConfig,
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
	kargoSvcCli svcv1alpha1connect.KargoServiceClient,
) error {
	res, err := kargoSvcCli.WatchClusterConfig(ctx, connect.NewRequest(&v1alpha1.WatchClusterConfigRequest{}))
	if err != nil {
		return fmt.Errorf("watch clusterconfig: %w", err)
	}
	defer func() {
		if conn, connErr := res.Conn(); connErr == nil {
			_ = conn.CloseRequest()
		}
	}()
	for {
		if !res.Receive() {
			if err = res.Err(); err != nil {
				return fmt.Errorf("watch clusterconfig: %w", err)
			}
			return errors.New("unexpected end of watch stream")
		}
		msg := res.Msg()
		if msg == nil || msg.ClusterConfig == nil {
			return errors.New("unexpected response")
		}
		token, ok := api.RefreshAnnotationValue(msg.ClusterConfig.GetAnnotations())
		if !ok {
			return fmt.Errorf(
				"ClusterConfig %q has no %q annotation",
				msg.ClusterConfig.Name, kargoapi.AnnotationKeyRefresh,
			)
		}
		if msg.ClusterConfig.Status.LastHandledRefresh == token {
			return nil
		}
	}
}
