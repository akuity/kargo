package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

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

// nolint: lll
// @id ListClusterPromotionTasks
// @Summary List ClusterPromotionTasks
// @Description List ClusterPromotionTask resources. Returns a
// @Description ClusterPromotionTaskList resource.
// @Tags Core, Shared, Cluster-Scoped Resource
// @Security BearerAuth
// @Produce json
// @Success 200 {object} object "ClusterPromotionTaskList custom resource (github.com/akuity/kargo/api/v1alpha1.ClusterPromotionTaskList)"
// @Router /v1beta1/shared/cluster-promotion-tasks [get]
func (s *server) listClusterPromotionTasks(c *gin.Context) {
	ctx := c.Request.Context()

	list := &kargoapi.ClusterPromotionTaskList{}
	if err := s.client.List(ctx, list); err != nil {
		_ = c.Error(err)
		return
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs kargoapi.ClusterPromotionTask) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	c.JSON(http.StatusOK, list)
}
