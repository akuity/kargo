package promotionrollouts

import (
	"net/http"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/promotionrolloutstore"
)

// @id GetPromotionRollout
// @Summary Retrieve a PromotionRollout
// @Description Retrieve one of a project's PromotionRollouts.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param promotion-rollout-name path string true "PromotionRollout name"
// @Produce json
// @Success 200 {object} kargoapi.PromotionRollout "PromotionRollout resource"
// @Router /v1beta1/projects/{project}/promotion-rollouts/{promotion-rollout-name} [get]
func (h *Handler) get(c *gin.Context) {
	ctx := c.Request.Context()
	project, name := c.Param(paramProject), c.Param(paramName)
	stored, err := h.store.Get(ctx, project, name)
	if err != nil {
		_ = c.Error(storeError(err, name))
		return
	}
	var rollouts []*kargoapi.PromotionRollout
	if rollouts, err = h.present(ctx, project, []promotionrolloutstore.Rollout{stored}); err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, rollouts[0])
}
