package targets

import (
	"net/http"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
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
func (h *Handler) create(c *gin.Context) {
	target, ok := bindTarget(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	project := c.Param(paramProject)

	row, err := rowFromTarget(target)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if row, err = h.store.Create(ctx, project, row); err != nil {
		_ = c.Error(storeError(err, target.Name))
		return
	}
	logChange(ctx, "Target created", project, row.Name)

	var created *kargoapi.Target
	if created, err = targetFromRow(row, project); err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, created)
}
