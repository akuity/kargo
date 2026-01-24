package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

// @id RefreshProjectConfig
// @Summary Refresh ProjectConfig
// @Description Refresh the single ProjectConfig resource in a project's
// @Description namespace. Refreshing enqueues the resource for reconciliation
// @Description by its corresponding controller.
// @Tags Core, Config, Project-Level, Singleton
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/config/refresh [post]
func (s *server) refreshProjectConfig(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")

	obj := &kargoapi.ProjectConfig{
		ObjectMeta: metav1.ObjectMeta{Name: project, Namespace: project},
	}

	if err := api.RefreshObject(ctx, s.client.InternalClient(), obj); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}
