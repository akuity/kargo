package targets

import (
	"net/http"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// @id GetTarget
// @Summary Retrieve a Target
// @Description Retrieve one of a project's Targets.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param target-name path string true "Target name"
// @Produce json
// @Success 200 {object} kargoapi.Target "Target resource"
// @Router /v1beta1/projects/{project}/targets/{target-name} [get]
func (h *Handler) get(c *gin.Context) {
	name := c.Param(paramName)
	var (
		target *kargoapi.Target
		err    error
	)
	if target, err = h.store.GetTarget(c.Request.Context(), c.Param(paramProject), name); err != nil {
		_ = c.Error(storeError(err, name))
		return
	}
	c.JSON(http.StatusOK, target)
}
