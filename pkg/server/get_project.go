package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// @id GetProject
// @Summary Retrieve a Project resource
// @Description Retrieve a Project resource.
// @Tags Core, Cluster-Scoped Resource
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Success 200 {object} kargoapi.Project "Project custom resource (github.com/akuity/kargo/api/v1alpha1.Project)"
// @Router /v1beta1/projects/{project} [get]
func (s *server) getProject(c *gin.Context) {
	ctx := c.Request.Context()

	name := c.Param("project")

	project := &kargoapi.Project{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: name},
		project,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, project)
}
