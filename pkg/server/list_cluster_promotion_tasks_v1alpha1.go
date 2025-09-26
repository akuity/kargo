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

func (s *server) ListClusterPromotionTasks(
	ctx context.Context,
	_ *connect.Request[svcv1alpha1.ListClusterPromotionTasksRequest],
) (*connect.Response[svcv1alpha1.ListClusterPromotionTasksResponse], error) {
	var list kargoapi.ClusterPromotionTaskList
	if err := s.client.List(ctx, &list); err != nil {
		return nil, fmt.Errorf("list clusterpromotiontasklist: %w", err)
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs kargoapi.ClusterPromotionTask) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	cats := make([]*kargoapi.ClusterPromotionTask, len(list.Items))
	for idx := range list.Items {
		cats[idx] = &list.Items[idx]
	}

	return connect.NewResponse(&svcv1alpha1.ListClusterPromotionTasksResponse{
		ClusterPromotionTasks: cats,
	}), nil
}
