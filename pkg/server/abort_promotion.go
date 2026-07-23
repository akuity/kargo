package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

// @id AbortPromotion
// @Summary Abort a Promotion
// @Description Abort a running Promotion.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param promotion path string true "Promotion name"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/promotions/{promotion}/abort [post]
func (s *server) abortPromotion(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	name := c.Param("promotion")

	objKey := client.ObjectKey{
		Namespace: project,
		Name:      name,
	}
	if err := api.AbortPromotion(ctx, s.client, objKey, kargoapi.AbortActionTerminate); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}
