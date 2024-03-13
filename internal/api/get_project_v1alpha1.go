package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) GetProject(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetProjectRequest],
) (*connect.Response[svcv1alpha1.GetProjectResponse], error) {
	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	var project kargoapi.Project
	if err := s.client.Get(
		ctx, client.ObjectKey{
			Name: name,
		},
		&project,
	); err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}

	return connect.NewResponse(
		&svcv1alpha1.GetProjectResponse{
			Project: &project,
		},
	), nil
}
