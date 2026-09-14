package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// @id GetTarget
// @Summary Retrieve a Target
// @Description Retrieve a Target resource from a project's namespace.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param target path string true "Target name"
// @Produce json
// @Success 200 {object} kargoapi.Target "Target custom resource"
// @Router /v1beta1/projects/{project}/targets/{target} [get]
func (s *server) getTarget(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	name := c.Param("target")

	target := &kargoapi.Target{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: name, Namespace: project},
		target,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, target)
}
