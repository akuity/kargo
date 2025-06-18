package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (s *server) DeleteProjectConfig(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.DeleteProjectConfigRequest],
) (*connect.Response[svcv1alpha1.DeleteProjectConfigResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	if err := s.client.Delete(
		ctx,
		&kargoapi.ProjectConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      project,
				Namespace: project,
			},
		},
	); err != nil {
		return nil, fmt.Errorf("delete project config: %w", err)
	}
	return connect.NewResponse(&svcv1alpha1.DeleteProjectConfigResponse{}), nil
}
