package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// nolint: lll
// @id GetClusterPromotionTask
// @Summary Retrieve a ClusterPromotionTask
// @Description Retrieve a ClusterPromotionTask by name.
// @Tags Core, Shared, Cluster-Scoped Resource
// @Security BearerAuth
// @Param cluster-promotion-task path string true "ClusterPromotionTask name"
// @Produce json
// @Success 200 {object} kargoapi.ClusterPromotionTask "ClusterPromotionTask custom resource (github.com/akuity/kargo/api/v1alpha1.ClusterPromotionTask)"
// @Router /v1beta1/shared/cluster-promotion-tasks/{cluster-promotion-task} [get]
func (s *server) getClusterPromotionTask(c *gin.Context) {
	ctx := c.Request.Context()

	name := c.Param("cluster-promotion-task")

	task := &kargoapi.ClusterPromotionTask{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: name},
		task,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, task)
}
