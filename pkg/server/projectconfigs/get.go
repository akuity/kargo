package projectconfigs

import (
	"net/http"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/projectconfigstore"
)

// This keeps the kargoapi import above in scope for the @Success annotation
// on get, which documents the response type without constructing one
// directly (projectconfigstore.ToAPI produces it).
var _ kargoapi.ProjectConfig

// @id GetProjectConfig
// @Summary Retrieve ProjectConfig
// @Description Retrieve a project's ProjectConfig.
// @Tags Core, Project-Level, Config, Singleton
// @Security BearerAuth
// @Param project path string true "Project name"
// @Produce json
// @Success 200 {object} kargoapi.ProjectConfig "ProjectConfig resource"
// @Router /v1beta1/projects/{project}/config [get]
func (h *Handler) get(c *gin.Context) {
	if c.Query("watch") == "true" {
		_ = c.Error(errWatchNotImplemented)
		return
	}
	project, ok := projectOf(c)
	if !ok {
		return
	}
	cfg, err := h.store.Get(c.Request.Context(), project.Name, string(project.UID))
	if err != nil {
		_ = c.Error(storeError(err, project.Name))
		return
	}
	c.JSON(http.StatusOK, projectconfigstore.ToAPI(cfg))
}
