package targets

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// @id DeleteTarget
// @Summary Delete a Target
// @Description Delete one of a project's Targets.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param target-name path string true "Target name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/projects/{project}/targets/{target-name} [delete]
func (h *Handler) delete(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param(paramProject)
	name := c.Param(paramName)

	if err := h.store.Delete(ctx, project, name); err != nil {
		_ = c.Error(storeError(err, name))
		return
	}
	logChange(ctx, "Target deleted", project, name)

	c.Status(http.StatusNoContent)
}
