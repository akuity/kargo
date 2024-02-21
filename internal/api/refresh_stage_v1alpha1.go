package api

import (
	"context"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) RefreshStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.RefreshStageRequest],
) (*connect.Response[svcv1alpha1.RefreshStageResponse], error) {
	if err := validateProjectAndStageNonEmpty(req.Msg.GetProject(), req.Msg.GetName()); err != nil {
		return nil, err
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}

	objKey := client.ObjectKey{
		Namespace: req.Msg.GetProject(),
		Name:      req.Msg.GetName(),
	}
	stage, err := kargoapi.RefreshStage(ctx, s.client, objKey)
	if err != nil {
		return nil, err
	}
	// If there is a current promotion then refresh it too.
	if stage.Status.CurrentPromotion != nil {
		if _, err := kargoapi.RefreshPromotion(ctx, s.client, client.ObjectKey{
			Namespace: req.Msg.GetProject(),
			Name:      stage.Status.CurrentPromotion.Name,
		}); err != nil {
			return nil, err
		}
	}
	return connect.NewResponse(&svcv1alpha1.RefreshStageResponse{
		Stage: stage,
	}), nil
}
