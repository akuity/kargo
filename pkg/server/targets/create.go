package targets

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

// @id CreateTarget
// @Summary Create a Target
// @Description Create a Target in a project. Targets are managed through
// @Description this API rather than as Kubernetes resources. The body's
// @Description metadata.name is required; its metadata.namespace, if given,
// @Description must be the project. Status and server-managed metadata are
// @Description ignored.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param project path string true "Project name"
// @Param body body kargoapi.Target true "Target resource"
// @Success 201 {object} kargoapi.Target "Target resource"
// @Router /v1beta1/projects/{project}/targets [post]
func (h *Handler) create(c *gin.Context, target *kargoapi.Target) {
	ctx := c.Request.Context()
	project := c.Param(paramProject)

	created, err := h.store.CreateTarget(ctx, project, target)
	if err != nil {
		if errors.Is(err, database.ErrProjectNotMirrored) {
			c.Header("Retry-After", "1")
		}
		_ = c.Error(storeError(err, target.Name))
		return
	}
	logChange(ctx, "Target created", project, created.Name)

	c.JSON(http.StatusCreated, created)
}
