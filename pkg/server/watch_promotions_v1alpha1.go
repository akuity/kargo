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

func (s *server) WatchPromotions(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.WatchPromotionsRequest],
	stream *connect.ServerStream[svcv1alpha1.WatchPromotionsResponse],
) error {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return err
	}

	stage := req.Msg.GetStage()

	if stage != "" {
		if err := s.client.Get(ctx, client.ObjectKey{
			Namespace: project,
			Name:      stage,
		}, &kargoapi.Stage{}); err != nil {
			return fmt.Errorf("get stage: %w", err)
		}
	}

	w, err := s.client.Watch(ctx, &kargoapi.PromotionList{}, client.InNamespace(project))
	if err != nil {
		return fmt.Errorf("watch promotion: %w", err)
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
			promotion, ok := e.Object.(*kargoapi.Promotion)
			if !ok {
				return fmt.Errorf("unexpected object type %T", e.Object)
			}
			// FIXME: Current (dynamic) client doesn't support filtering with indexed field by indexer,
			// so manually filter stage here.
			if stage != "" && stage != promotion.Spec.Stage {
				continue
			}
			if err = stream.Send(&svcv1alpha1.WatchPromotionsResponse{
				Promotion: promotion,
				Type:      string(e.Type),
			}); err != nil {
				return fmt.Errorf("send response: %w", err)
			}
		}
	}
}
