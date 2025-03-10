package server

import (
	"context"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/internal/api"
)

func (s *server) RefreshStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.RefreshStageRequest],
) (*connect.Response[svcv1alpha1.RefreshStageResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}
	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	objKey := client.ObjectKey{
		Namespace: project,
		Name:      name,
	}
	stage, err := api.RefreshStage(ctx, s.client, objKey)
	if err != nil {
		return nil, err
	}
	// If there is a current promotion then refresh it too. Do this with the API
	// server's own internal client so that individual users are not required to
	// have this permission, which they really do not otherwise need.
	if stage.Status.CurrentPromotion != nil {
		if _, err := api.RefreshPromotion(ctx, s.client.InternalClient(), client.ObjectKey{
			Namespace: project,
			Name:      stage.Status.CurrentPromotion.Name,
		}); err != nil {
			return nil, err
		}
	}
	return connect.NewResponse(&svcv1alpha1.RefreshStageResponse{
		Stage: stage,
	}), nil
}
