package server

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// nolint: lll
// @id ListClusterPromotionTasks
// @Summary List ClusterPromotionTasks
// @Description List ClusterPromotionTask resources. Returns a
// @Description ClusterPromotionTaskList resource.
// @Tags Core, Shared, Cluster-Scoped Resource
// @Security BearerAuth
// @Produce json
// @Success 200 {object} kargoapi.ClusterPromotionTaskList "ClusterPromotionTaskList custom resource (github.com/akuity/kargo/api/v1alpha1.ClusterPromotionTaskList)"
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
