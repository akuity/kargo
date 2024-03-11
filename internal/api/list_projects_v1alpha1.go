package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/akuity/kargo/api/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) ListProjects(
	ctx context.Context,
	_ *connect.Request[svcv1alpha1.ListProjectsRequest],
) (*connect.Response[svcv1alpha1.ListProjectsResponse], error) {
	projects := &kargoapi.ProjectList{}
	if err := s.client.List(ctx, projects); err != nil {
		return nil, fmt.Errorf("error listing Projects: %w", err)
	}
	projectProtos := make([]*v1alpha1.Project, len(projects.Items))
	for i := range projects.Items {
		projectProtos[i] = &projects.Items[i]
	}
	return connect.NewResponse(&svcv1alpha1.ListProjectsResponse{
		Projects: projectProtos,
	}), nil
}
