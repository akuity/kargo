package api

import (
	"context"
	"fmt"
	"sort"

	"connectrpc.com/connect"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) ListDetailedProjects(
	ctx context.Context,
	_ *connect.Request[svcv1alpha1.ListDetailedProjectsRequest],
) (*connect.Response[svcv1alpha1.ListDetailedProjectsResponse], error) {
	var list kargoapi.ProjectList
	if err := s.client.List(ctx, &list); err != nil {
		return nil, fmt.Errorf("error listing Projects: %w", err)
	}

	sort.Slice(list.Items, func(i, j int) bool {
		return list.Items[i].Name < list.Items[j].Name
	})

	projects := make([]*svcv1alpha1.DetailedProject, len(list.Items))
	for i := range list.Items {
		var stages kargoapi.StageList
		if err := s.client.List(ctx, &stages); err != nil {
			return nil, fmt.Errorf("error listing Stages: %w", err)
		}
		proj := &svcv1alpha1.DetailedProject{
			Project: &list.Items[i],
			Stages:  make([]*kargoapi.Stage, len(stages.Items)),
		}
		for j := range stages.Items {
			proj.Stages[j] = &stages.Items[j]
		}
		projects[i] = proj
	}
	return connect.NewResponse(&svcv1alpha1.ListDetailedProjectsResponse{
		DetailedProjects: projects,
	}), nil
}
