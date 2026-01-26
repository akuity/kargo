package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	libClient "sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

func (s *server) WatchWarehouses(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchWarehousesRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchWarehousesResponse],
) error {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return err
	}

	name := req.Msg.GetName()

	if name != "" {
		if err := s.client.Get(ctx, libClient.ObjectKey{
			Namespace: project,
			Name:      name,
		}, &kargoapi.Warehouse{}); err != nil {
			return fmt.Errorf("get warehouse: %w", err)
		}
	}

	watchOpts := []libClient.ListOption{libClient.InNamespace(project)}
	if name != "" {
		watchOpts = append(watchOpts, libClient.MatchingFields{"metadata.name": name})
	}
	w, err := s.client.Watch(ctx, &kargoapi.WarehouseList{}, watchOpts...)
	if err != nil {
		return fmt.Errorf("watch warehouse: %w", err)
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
			warehouse, ok := e.Object.(*kargoapi.Warehouse)
			if !ok {
				return fmt.Errorf("unexpected object type %T", e.Object)
			}
			if err := stream.Send(&svcv1alpha1.WatchWarehousesResponse{
				Warehouse: warehouse,
				Type:      string(e.Type),
			}); err != nil {
				return fmt.Errorf("send response: %w", err)
			}
		}
	}
}
