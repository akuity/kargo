package server

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (s *server) ListPromotionTasks(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListPromotionTasksRequest],
) (*connect.Response[svcv1alpha1.ListPromotionTasksResponse], error) {
	project := req.Msg.GetProject()

	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	var list kargoapi.PromotionTaskList

	opts := []client.ListOption{
		client.InNamespace(project),
	}

	if err := s.client.List(ctx, &list, opts...); err != nil {
		return nil, fmt.Errorf("list promotiontasks: %w", err)
	}

	slices.SortFunc(list.Items, func(lhs, rhs kargoapi.PromotionTask) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	pts := make([]*kargoapi.PromotionTask, len(list.Items))
	for idx := range list.Items {
		pts[idx] = &list.Items[idx]
	}

	return connect.NewResponse(&svcv1alpha1.ListPromotionTasksResponse{
		PromotionTasks: pts,
	}), nil
}
