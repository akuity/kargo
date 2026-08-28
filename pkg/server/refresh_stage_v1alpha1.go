package server

import (
	"context"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
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
	if err := s.authorizeFn(
		ctx, "get", kargoapi.GroupVersion.WithResource("stages"), "", objKey,
	); err != nil {
		return nil, err
	}
	stage, err := api.RefreshStage(ctx, s.client.InternalClient(), objKey)
	if err != nil {
		return nil, err
	}
	// If there is a current promotion then refresh it too. The actual patch is
	// still done with the API server's own internal client so that individual
	// users are not required to have patch permission, which they really do
	// not otherwise need -- but the caller must still be able to "get" it.
	if stage.Status.CurrentPromotion != nil {
		promoKey := client.ObjectKey{
			Namespace: project,
			Name:      stage.Status.CurrentPromotion.Name,
		}
		if err := s.authorizeFn(
			ctx, "get", kargoapi.GroupVersion.WithResource("promotions"), "", promoKey,
		); err != nil {
			return nil, err
		}
		if _, err := api.RefreshPromotion(ctx, s.client.InternalClient(), promoKey); err != nil {
			return nil, err
		}
	}
	return connect.NewResponse(&svcv1alpha1.RefreshStageResponse{
		Stage: stage,
	}), nil
}
