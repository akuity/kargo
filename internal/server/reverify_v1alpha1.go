package server

import (
	"context"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/internal/api"
)

func (s *server) Reverify(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ReverifyRequest],
) (*connect.Response[svcv1alpha1.ReverifyResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}
	stage := req.Msg.GetStage()
	if err := validateFieldNotEmpty("stage", stage); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	objKey := client.ObjectKey{
		Namespace: project,
		Name:      stage,
	}
	if err := api.ReverifyStageFreight(ctx, s.client, objKey); err != nil {
		return nil, err
	}
	return connect.NewResponse(&svcv1alpha1.ReverifyResponse{}), nil
}
