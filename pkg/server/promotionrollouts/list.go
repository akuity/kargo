package promotionrollouts

import (
	"net/http"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/promotionrolloutstore"
)

// @id ListPromotionRollouts
// @Summary List PromotionRollouts
// @Description List a project's PromotionRollouts, optionally narrowed to
// @Description those of a particular Stage, in creation order. Returns a
// @Description PromotionRolloutList resource.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param stage query string false "Only PromotionRollouts of this Stage"
// @Produce json
// @Success 200 {object} kargoapi.PromotionRolloutList "PromotionRolloutList resource"
// @Router /v1beta1/projects/{project}/promotion-rollouts [get]
func (h *Handler) list(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param(paramProject)
	stage := c.Query("stage")

	stored, err := h.store.List(ctx, project, stage)
	if err != nil {
		_ = c.Error(err)
		return
	}
	rollouts := make([]*kargoapi.PromotionRollout, len(stored))
	for i, s := range stored {
		rollouts[i] = promotionrolloutstore.ToAPI(s)
	}
	if err = h.fillTargetStatus(ctx, project, stage, rollouts); err != nil {
		_ = c.Error(err)
		return
	}
	list := &kargoapi.PromotionRolloutList{Items: make([]kargoapi.PromotionRollout, len(rollouts))}
	for i, rollout := range rollouts {
		list.Items[i] = *rollout
	}
	c.JSON(http.StatusOK, list)
}
