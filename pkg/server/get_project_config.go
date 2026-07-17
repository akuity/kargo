package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

// @id GetProjectConfig
// @Summary Retrieve ProjectConfig
// @Description Retrieve the single ProjectConfig resource from a project's
// @Description namespace.
// @Tags Core, Project-Level, Config, Singleton
// @Security BearerAuth
// @Param project path string true "Project name"
// @Produce json
// @Success 200 {object} kargoapi.ProjectConfig "ProjectConfig custom resource"
// @Router /v1beta1/projects/{project}/config [get]
func (s *server) getProjectConfig(c *gin.Context) {
	ctx := c.Request.Context()

	project := c.Param("project")

	if watchMode := c.Query("watch") == trueStr; watchMode {
		s.watchProjectConfig(c, project)
		return
	}

	config := &kargoapi.ProjectConfig{}
	if err := s.client.Get(
		ctx, client.ObjectKey{Name: project, Namespace: project}, config,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, config)
}

func (s *server) watchProjectConfig(c *gin.Context, project string) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	// Validate that the ProjectConfig exists before starting the watch
	config := &kargoapi.ProjectConfig{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: project, Namespace: project},
		config,
	); err != nil {
		_ = c.Error(err)
		return
	}

	// ProjectConfig is namespaced, namespace = project
	w, err := s.client.Watch(
		ctx,
		&kargoapi.ProjectConfigList{},
		client.InNamespace(project),
		client.MatchingFields{"metadata.name": project},
	)
	if err != nil {
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch project config: %w", err))
		return
	}
	defer w.Stop()

	keepaliveTicker := time.NewTicker(30 * time.Second)
	defer keepaliveTicker.Stop()

	SetSSEHeaders(c)

	for {
		select {
		case <-ctx.Done():
			logger.Debug("watch context done", "error", ctx.Err())
			return

		case <-keepaliveTicker.C:
			if !WriteSSEKeepalive(c) {
				return
			}

		case e, ok := <-w.ResultChan():
			if !ok {
				logger.Debug("watch channel closed")
				return
			}
			if !ConvertAndSendWatchEvent(c, e, (*kargoapi.ProjectConfig)(nil)) {
				return
			}
		}
	}
}
