package server

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (s *server) ListProjects(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListProjectsRequest],
) (*connect.Response[svcv1alpha1.ListProjectsResponse], error) {
	var list kargoapi.ProjectList
	if err := s.client.List(ctx, &list); err != nil {
		return nil, fmt.Errorf("error listing Projects: %w", err)
	}

	slices.SortFunc(list.Items, func(a, b kargoapi.Project) int {
		return strings.Compare(a.Name, b.Name)
	})

	var filtered []kargoapi.Project
	if req.Msg.GetFilter() != "" {
		filter := strings.ToLower(req.Msg.GetFilter())
		for i := 0; i < len(list.Items); i++ {
			if strings.Contains(strings.ToLower(list.Items[i].Name), filter) {
				filtered = append(filtered, list.Items[i])
			}
		}
		list.Items = filtered
	}

	total := len(list.Items)
	pageSize := len(list.Items)
	if req.Msg.GetPageSize() > 0 {
		pageSize = int(req.Msg.GetPageSize())
	}

	start := int(req.Msg.GetPage()) * pageSize
	end := start + pageSize

	if start >= len(list.Items) {
		return connect.NewResponse(&svcv1alpha1.ListProjectsResponse{}), nil
	}

	if end > len(list.Items) {
		end = len(list.Items)
	}

	list.Items = list.Items[start:end]
	projects := make([]*kargoapi.Project, len(list.Items))
	for i := range list.Items {
		projects[i] = &list.Items[i]
	}
	return connect.NewResponse(&svcv1alpha1.ListProjectsResponse{
		Projects: projects,
		Total:    int32(total), // nolint: gosec
	}), nil
}
