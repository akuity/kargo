package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// @id GetPromotionTask
// @Summary Retrieve a PromotionTask
// @Description Retrieve a PromotionTask resource from a project's namespace.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param promotion-task path string true "PromotionTask name"
// @Success 200 {object} kargoapi.PromotionTask "PromotionTask custom resource"
// @Router /v1beta1/projects/{project}/promotion-tasks/{promotion-task} [get]
func (s *server) getPromotionTask(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("promotion-task")

	task := &kargoapi.PromotionTask{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: name, Namespace: project},
		task,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, task)
}
