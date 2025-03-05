package server

import (
	"context"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) AbortPromotion(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.AbortPromotionRequest],
) (*connect.Response[svcv1alpha1.AbortPromotionResponse], error) {
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
	if err := api.AbortPromotion(ctx, s.client, objKey, kargoapi.AbortActionTerminate); err != nil {
		return nil, err
	}
	return connect.NewResponse(&svcv1alpha1.AbortPromotionResponse{}), nil
}
