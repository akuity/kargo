package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// @id DeleteProjectConfig
// @Summary Delete a ProjectConfig resource
// @Description Delete the single ProjectConfig resource from a project's
// @Description namespace.
// @Tags Core, Project-Level, Config, Singleton
// @Security BearerAuth
// @Param project path string true "Project name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/projects/{project}/config [delete]
func (s *server) deleteProjectConfig(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")

	if err := s.client.Delete(
		ctx,
		&kargoapi.ProjectConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      project,
				Namespace: project,
			},
		},
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
