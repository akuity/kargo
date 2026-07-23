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

// @id GetStage
// @Summary Retrieve a Stage
// @Description Retrieve a Stage resource from a project's namespace.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Success 200 {object} kargoapi.Stage "Stage custom resource (github.com/akuity/kargo/api/v1alpha1.Stage)"
// @Router /v1beta1/projects/{project}/stages/{stage} [get]
func (s *server) getStage(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	name := c.Param("stage")

	if watchMode := c.Query("watch") == trueStr; watchMode {
		s.watchStage(c, project, name)
		return
	}

	stage := &kargoapi.Stage{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: name, Namespace: project},
		stage,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, stage)
}

func (s *server) watchStage(c *gin.Context, project, name string) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	// Validate that the Stage exists before starting the watch
	stage := &kargoapi.Stage{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: name, Namespace: project},
		stage,
	); err != nil {
		_ = c.Error(err)
		return
	}

	w, err := s.client.Watch(
		ctx,
		&kargoapi.StageList{},
		client.InNamespace(project),
		client.MatchingFields{"metadata.name": name},
	)
	if err != nil {
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch stage: %w", err))
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
			if !ConvertAndSendWatchEvent(c, e, (*kargoapi.Stage)(nil)) {
				return
			}
		}
	}
}
