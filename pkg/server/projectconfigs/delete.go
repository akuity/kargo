package projectconfigs

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/auth/can"
)

// @id DeleteProjectConfig
// @Summary Delete a ProjectConfig resource
// @Description Delete a project's ProjectConfig.
// @Tags Core, Project-Level, Config, Singleton
// @Security BearerAuth
// @Param project path string true "Project name"
// @Success 204 "Deleted successfully"
// @Router /v1beta1/projects/{project}/config [delete]
func (h *Handler) delete(c *gin.Context) {
	project, ok := projectOf(c)
	if !ok {
		return
	}
	if err := h.Delete(c.Request.Context(), project); err != nil {
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Delete deletes the Project's ProjectConfig. The caller must hold the
// delete verb on it.
func (h *Handler) Delete(ctx context.Context, project *kargoapi.Project) error {
	if err := h.authorize(ctx, can.Delete().ProjectConfig(project.Name)); err != nil {
		return err
	}
	if err := h.store.Delete(ctx, project.Name, string(project.UID)); err != nil {
		return storeError(err, project.Name)
	}
	logChange(ctx, "ProjectConfig deleted", project.Name)
	return nil
}
