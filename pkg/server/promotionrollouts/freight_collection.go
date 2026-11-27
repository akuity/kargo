package promotionrollouts

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/auth/can"
)

// @id GetPromotionRolloutFreightCollection
// @Summary Retrieve the FreightCollection a PromotionRollout recorded
// @Description Retrieve the FreightCollection a PromotionRollout recorded when
// @Description it started: the Freight it promotes and, for each other origin
// @Description its Stage requests Freight from, the Freight carried forward
// @Description from the Stage's previous rollout. The rollout is named by its
// @Description id, as a Promotion it created names it. A rollout that has not
// @Description started, or that recorded no collection, has an empty one.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param id path string true "PromotionRollout id"
// @Produce json
// @Success 200 {object} kargoapi.FreightCollection "FreightCollection"
// @Router /v1beta1/projects/{project}/promotion-rollouts/{id}/freight-collection [get]
func (h *Handler) getFreightCollection(c *gin.Context) {
	ctx := c.Request.Context()
	project, raw := c.Param(paramProject), c.Param(paramID)
	// Only the rollout is read, so getting it is all the caller needs: unlike
	// get, nothing is reported of the Promotions it created.
	if err := h.authorize(ctx, can.Get().PromotionRollout(project, "")); err != nil {
		_ = c.Error(err)
		return
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		_ = c.Error(libhttp.ErrorStr(fmt.Sprintf("invalid id %q", raw), http.StatusBadRequest))
		return
	}
	var collection *kargoapi.FreightCollection
	if collection, err = h.store.GetFreightCollection(ctx, project, id); err != nil {
		_ = c.Error(storeError(err, raw))
		return
	}
	c.JSON(http.StatusOK, collection)
}
