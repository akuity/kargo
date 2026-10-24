package targets

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	libhttp "github.com/akuity/kargo/pkg/http"
)

// defaultPromotionsLimit and maxPromotionsLimit bound how much of a Target's
// Promotion history one request returns. The history is never pruned.
const (
	defaultPromotionsLimit = 100
	maxPromotionsLimit     = 1000
)

// TargetPromotionList is a list of Promotions to a Target, newest first.
type TargetPromotionList struct {
	Items []TargetPromotion `json:"items"`
} // @name TargetPromotionList

// @id ListTargetPromotions
// @Summary List a Target's Promotions
// @Description List the Promotions to one of a project's Targets, newest
// @Description first, from every Stage, including those Kubernetes no longer
// @Description has. Returns a TargetPromotionList.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param target-name path string true "Target name"
// @Param limit query int false "Most Promotions to return, 1 to 1000; default 100"
// @Produce json
// @Success 200 {object} TargetPromotionList "TargetPromotionList"
// @Router /v1beta1/projects/{project}/targets/{target-name}/promotions [get]
func (h *Handler) listPromotions(c *gin.Context) {
	limit, err := parseLimit(c.Query("limit"))
	if err != nil {
		_ = c.Error(err)
		return
	}
	rows, err := h.store.Promotions(
		c.Request.Context(),
		c.Param(paramProject),
		c.Param(paramName),
		limit,
	)
	if err != nil {
		_ = c.Error(err)
		return
	}
	promotions := make([]TargetPromotion, 0, len(rows))
	for _, row := range rows {
		promotions = append(promotions, *targetPromotion(row))
	}
	c.JSON(http.StatusOK, &TargetPromotionList{Items: promotions})
}

// parseLimit parses the limit query parameter. An empty value is the default.
func parseLimit(raw string) (int32, error) {
	if raw == "" {
		return defaultPromotionsLimit, nil
	}
	limit, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || limit < 1 || limit > maxPromotionsLimit {
		return 0, libhttp.Error(
			fmt.Errorf("limit must be an integer from 1 to %d", maxPromotionsLimit),
			http.StatusBadRequest,
		)
	}
	return int32(limit), nil
}
