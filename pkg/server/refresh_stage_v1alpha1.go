package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

// @id RefreshStage
// @Summary Refresh a Stage
// @Description Refresh a Stage resource in a project's namespace. Refreshing
// @Description enqueues the resource for reconciliation by its corresponding
// @Description controller.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Success 200 "Success"
// @Router /v1beta1/projects/{project}/stages/{stage}/refresh [post]
func (s *server) refreshStage(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")
	stageName := c.Param("stage")

	stage := &kargoapi.Stage{
		ObjectMeta: metav1.ObjectMeta{Name: stageName, Namespace: project},
	}

	if err := api.RefreshObject(ctx, s.client.InternalClient(), stage); err != nil {
		_ = c.Error(err)
		return
	}

	// If there is a current Promotion then refresh it, too
	if stage.Status.CurrentPromotion != nil {
		promo := &kargoapi.Promotion{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: project,
				Name:      stage.Status.CurrentPromotion.Name,
			},
		}
		if err := api.RefreshObject(ctx, s.client.InternalClient(), promo); err != nil {
			_ = c.Error(fmt.Errorf("failed to refresh current Promotion: %w", err))
			return
		}
	}

	c.Status(http.StatusOK)
}
