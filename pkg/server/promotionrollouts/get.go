package promotionrollouts

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/promotionrolloutstore"
	libhttp "github.com/akuity/kargo/pkg/http"
)

// @id GetPromotionRollout
// @Summary Retrieve a PromotionRollout
// @Description Retrieve one of a Stage's PromotionRollouts by its number
// @Description among them.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Param number path int64 true "PromotionRollout number"
// @Produce json
// @Success 200 {object} kargoapi.PromotionRollout "PromotionRollout resource"
// @Router /v1beta1/projects/{project}/stages/{stage}/promotion-rollouts/{number} [get]
func (h *Handler) get(c *gin.Context) {
	ctx := c.Request.Context()
	project, stage, raw := c.Param(paramProject), c.Param(paramStage), c.Param(paramNumber)
	number, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || number <= 0 {
		_ = c.Error(libhttp.ErrorStr(fmt.Sprintf("invalid number %q", raw), http.StatusBadRequest))
		return
	}
	stored, err := h.store.GetByNumber(ctx, project, stage, number)
	if err != nil {
		_ = c.Error(storeError(err, fmt.Sprintf("%s #%d", stage, number)))
		return
	}
	var rollouts []*kargoapi.PromotionRollout
	if rollouts, err = h.present(ctx, project, []promotionrolloutstore.Rollout{stored}); err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, rollouts[0])
}
