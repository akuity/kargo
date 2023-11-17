package refresh

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/cli/option"
	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

func newRefreshStageCommand(opt *option.Option) *cobra.Command {
	var flag Flags
	cmd := &cobra.Command{
		Use:     "stage (STAGE)",
		Args:    option.ExactArgs(1),
		Example: "kargo refresh stage --project=guestbook (STAGE)",
		RunE:    refreshObject(opt, &flag, "stage"),
	}
	addRefreshFlags(cmd, &flag)
	option.OptionalProject(opt.Project)(cmd.Flags())
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
		return errors.Wrap(err, "watch stage")
	}
	defer func() {
		if conn, connErr := res.Conn(); connErr == nil {
			_ = conn.CloseRequest()
		}
	}()
	for {
		if !res.Receive() {
			if err = res.Err(); err != nil {
				return errors.Wrap(err, "watch stage")
			}
			return errors.New("unexpected end of watch stream")
		}
		msg := res.Msg()
		if msg == nil || msg.Stage == nil {
			return errors.New("unexpected response")
		}
		if msg.Stage.Metadata.Annotations == nil ||
			msg.Stage.Metadata.Annotations[kargoapi.AnnotationKeyRefresh] == "" {
			return nil
		}
	}
}
