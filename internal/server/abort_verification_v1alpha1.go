package server

import (
	"context"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) AbortVerification(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.AbortVerificationRequest],
) (*connect.Response[svcv1alpha1.AbortVerificationResponse], error) {
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
	if err := kargoapi.AbortStageFreightVerification(ctx, s.client, objKey); err != nil {
		return nil, err
	}
	return connect.NewResponse(&svcv1alpha1.AbortVerificationResponse{}), nil
}
