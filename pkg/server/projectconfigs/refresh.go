package projectconfigs

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/akuity/kargo/pkg/server/projectconfigs/wire"
)

// @id RefreshProjectConfig
// @Summary Refresh ProjectConfig
// @Description Request a reconcile of a project's ProjectConfig. The response
// @Description carries a token, which the ProjectConfig's
// @Description status.lastHandledRefresh reports once the refresh has been
// @Description handled.
// @Tags Core, Config, Project-Level, Singleton
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Success 200 {object} wire.RefreshResponse "Refresh requested"
// @Router /v1beta1/projects/{project}/config/refresh [post]
func (h *Handler) refresh(c *gin.Context) {
	project, ok := projectOf(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	token, err := h.store.RequestRefresh(ctx, project.Name, string(project.UID))
	if err != nil {
		_ = c.Error(storeError(err, project.Name))
		return
	}
	logChange(ctx, "ProjectConfig refresh requested", project.Name)
	c.JSON(http.StatusOK, wire.RefreshResponse{Token: token})
}
