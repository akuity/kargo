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

// @id GetPromotion
// @Summary Retrieve a Promotion
// @Description Retrieve a Promotion resource from a project's namespace.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param promotion path string true "Promotion name"
// @Produce json
// @Success 200 {object} kargoapi.Promotion "Promotion custom resource (github.com/akuity/kargo/api/v1alpha1.Promotion)"
// @Router /v1beta1/projects/{project}/promotions/{promotion} [get]
func (s *server) getPromotion(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")
	name := c.Param("promotion")

	if watchMode := c.Query("watch") == trueStr; watchMode {
		s.watchPromotion(c, project, name)
		return
	}

	promotion := &kargoapi.Promotion{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: name, Namespace: project},
		promotion,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, promotion)
}

func (s *server) watchPromotion(c *gin.Context, project, name string) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	// Validate that the Promotion exists before starting the watch
	promotion := &kargoapi.Promotion{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: name, Namespace: project},
		promotion,
	); err != nil {
		_ = c.Error(err)
		return
	}

	w, err := s.client.Watch(
		ctx,
		&kargoapi.PromotionList{},
		client.InNamespace(project),
		client.MatchingFields{"metadata.name": name},
	)
	if err != nil {
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch promotion: %w", err))
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
			if !ConvertAndSendWatchEvent(c, e, (*kargoapi.Promotion)(nil)) {
				return
			}
		}
	}
}
