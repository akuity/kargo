package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (s *server) DeleteStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteStageRequest],
) (*connect.Response[svcv1alpha1.DeleteStageResponse], error) {
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

	if err := s.client.Delete(
		ctx,
		&kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: project,
				Name:      name,
			},
		},
	); err != nil {
		return nil, fmt.Errorf("delete stage: %w", err)
	}
	return connect.NewResponse(&svcv1alpha1.DeleteStageResponse{}), nil
}
