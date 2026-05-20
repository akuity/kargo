package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	libClient "sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/logging"
)

func (s *server) WatchStages(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchStagesRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchStagesResponse],
) error {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return err
	}

	name := req.Msg.GetName()
	warehouses := req.Msg.GetFreightOrigins()

	if name != "" {
		if err := s.client.Get(ctx, libClient.ObjectKey{
			Namespace: project,
			Name:      name,
		}, &kargoapi.Stage{}); err != nil {
			return fmt.Errorf("get stage: %w", err)
		}
	}

	var opts []libClient.ListOption
	if name != "" {
		opts = append(opts, libClient.MatchingFields{"metadata.name": name})
	}
	w, err := s.client.Watch(
		ctx,
		&kargoapi.StageList{},
		buildWatchListOptions(project, req.Msg.GetResourceVersion(), opts...)...,
	)
	if err != nil {
		return fmt.Errorf("watch stage: %w", err)
	}
	defer w.Stop()
	for {
		select {
		case <-ctx.Done():
			logger := logging.LoggerFromContext(ctx)
			logger.Debug(ctx.Err().Error())
			return nil
		case e, ok := <-w.ResultChan():
			if !ok {
				return nil
			}
			if err := errorFromWatchEvent(e); err != nil {
				return err
			}
			stage, ok := e.Object.(*kargoapi.Stage)
			if !ok {
				return fmt.Errorf("unexpected object type %T", e.Object)
			}
			if len(warehouses) > 0 && !api.StageMatchesAnyWarehouse(stage, warehouses) {
				continue
			}
			if err := stream.Send(&svcv1alpha1.WatchStagesResponse{
				Stage: stage,
				Type:  string(e.Type),
			}); err != nil {
				return fmt.Errorf("send response: %w", err)
			}
		}
	}
}
