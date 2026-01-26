package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

// @id RefreshPromotion
// @Summary Refresh a Promotion
// @Description Refresh a Promotion resource in a project's namespace.
// @Description Refreshing enqueues the resource for reconciliation by its
// @Description corresponding controller.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param promotion path string true "Promotion name"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/promotions/{promotion}/refresh [post]
func (s *server) refreshPromotion(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	promotionName := c.Param("promotion")

	obj := &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{Name: promotionName, Namespace: project},
	}

	if err := api.RefreshObject(ctx, s.client.InternalClient(), obj); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}
