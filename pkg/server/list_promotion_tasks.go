package server

import (
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// nolint: lll
// @id ListPromotionTasks
// @Summary List PromotionTasks
// @Description List PromotionTask resources from a project's namespace. Returns
// @Description a PromotionTaskList resource.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Produce json
// @Success 200 {object} kargoapi.PromotionTaskList "PromotionTaskList custom resource (github.com/akuity/kargo/api/v1alpha1.PromotionTaskList)"
// @Router /v1beta1/projects/{project}/promotion-tasks [get]
func (s *server) listPromotionTasks(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")

	list := &kargoapi.PromotionTaskList{}
	if err := s.client.List(
		ctx, list, client.InNamespace(project),
	); err != nil {
		_ = c.Error(err)
		return
	}

	// Sort ascending by name
	slices.SortFunc(list.Items, func(lhs, rhs kargoapi.PromotionTask) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	c.JSON(http.StatusOK, list)
}
