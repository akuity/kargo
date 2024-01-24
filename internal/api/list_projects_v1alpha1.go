package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/v1alpha1"
)

func (s *server) ListProjects(
	ctx context.Context,
	_ *connect.Request[svcv1alpha1.ListProjectsRequest],
) (*connect.Response[svcv1alpha1.ListProjectsResponse], error) {
	projects := &kargoapi.ProjectList{}
	if err := s.client.List(ctx, projects); err != nil {
		return nil, errors.Wrap(err, "error listing Projects")
	}
	projectProtos := make([]*v1alpha1.Project, len(projects.Items))
	for i, project := range projects.Items {
		projectProtos[i] = typesv1alpha1.ToProjectProto(project)
	}
	return connect.NewResponse(&svcv1alpha1.ListProjectsResponse{
		Projects: projectProtos,
	}), nil
}
