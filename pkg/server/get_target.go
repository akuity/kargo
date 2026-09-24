package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/akuity/kargo/pkg/database"
)

// @id GetTarget
// @Summary Retrieve a Target
// @Description Retrieve a Target resource from a project's namespace.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param target path string true "Target name"
// @Produce json
// @Success 200 {object} kargoapi.Target "Target custom resource"
// @Router /v1beta1/projects/{project}/targets/{target} [get]
func (s *server) getTarget(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	name := c.Param("target")

	if s.store == nil {
		_ = c.Error(errDatabaseNotConfigured)
		return
	}

	if err := s.authorizeStoreRead(ctx, "get", "targets", project, name); err != nil {
		_ = c.Error(err)
		return
	}

	row, err := s.store.GetTarget(ctx, database.GetTargetParams{ProjectName: project, Name: name})
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			_ = c.Error(storeNotFound("targets", name))
			return
		}
		_ = c.Error(fmt.Errorf("error getting Target %q in Project %q: %w", name, project, err))
		return
	}
	target, err := database.TargetFromRow(row, project)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, target)
}
