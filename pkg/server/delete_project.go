package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// @id DeleteProject
// @Summary Delete a Project
// @Description Delete a Project resource and its associated namespace.
// @Tags Core, Cluster-Scoped Resource
// @Security BearerAuth
// @Param project path string true "Project name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/projects/{project} [delete]
func (s *server) deleteProject(c *gin.Context) {
	ctx := c.Request.Context()

	name := c.Param("project")

	if err := s.client.Delete(
		ctx,
		&kargoapi.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		},
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
