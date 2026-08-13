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

// @id GetPromotionRequest
// @Summary Retrieve a PromotionRequest
// @Description Retrieve a PromotionRequest resource from a project's namespace.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param promotion-request path string true "PromotionRequest name"
// @Produce json
// @Success 200 {object} kargoapi.PromotionRequest "PromotionRequest custom resource"
// @Router /v1beta1/projects/{project}/promotion-requests/{promotion-request} [get]
func (s *server) getPromotionRequest(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	name := c.Param("promotion-request")

	if watchMode := c.Query("watch") == trueStr; watchMode {
		s.watchPromotionRequest(c, project, name)
		return
	}

	promotionRequest := &kargoapi.PromotionRequest{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: name, Namespace: project},
		promotionRequest,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, promotionRequest)
}

func (s *server) watchPromotionRequest(c *gin.Context, project, name string) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	// Validate that the PromotionRequest exists before starting the watch
	promotionRequest := &kargoapi.PromotionRequest{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: name, Namespace: project},
		promotionRequest,
	); err != nil {
		_ = c.Error(err)
		return
	}

	w, err := s.client.Watch(
		ctx,
		&kargoapi.PromotionRequestList{},
		client.InNamespace(project),
		client.MatchingFields{"metadata.name": name},
	)
	if err != nil {
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch promotion request: %w", err))
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
			if !ConvertAndSendWatchEvent(c, e, (*kargoapi.PromotionRequest)(nil)) {
				return
			}
		}
	}
}
