package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

func (s *server) WatchFreight(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchFreightRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchFreightResponse],
) error {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return err
	}

	w, err := s.client.Watch(ctx, &kargoapi.FreightList{}, client.InNamespace(project))
	if err != nil {
		return fmt.Errorf("watch freight: %w", err)
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
			freight, ok := e.Object.(*kargoapi.Freight)
			if !ok {
				return fmt.Errorf("unexpected object type %T", e.Object)
			}
			if err := stream.Send(&svcv1alpha1.WatchFreightResponse{
				Freight: freight,
				Type:    string(e.Type),
			}); err != nil {
				return fmt.Errorf("send response: %w", err)
			}
		}
	}
}
