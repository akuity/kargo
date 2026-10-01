package targets

import (
	"net/http"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// @id UpdateTarget
// @Summary Update a Target
// @Description Replace a Target's labels and params. The body's metadata.name,
// @Description if given, must match the name in the URL. A metadata.uid or
// @Description metadata.resourceVersion in the body is a precondition: the
// @Description update is refused if the Target no longer matches it.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param target-name path string true "Target name"
// @Param body body kargoapi.Target true "Target resource"
// @Success 200 {object} kargoapi.Target "Target resource"
// @Router /v1beta1/projects/{project}/targets/{target-name} [put]
func (h *Handler) update(c *gin.Context, target *kargoapi.Target) {
	ctx := c.Request.Context()
	project := c.Param(paramProject)

	updated, err := h.store.UpdateTarget(ctx, project, target)
	if err != nil {
		_ = c.Error(storeError(err, target.Name))
		return
	}
	logChange(ctx, "Target updated", project, target.Name)

	c.JSON(http.StatusOK, updated)
}
