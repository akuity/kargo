package server

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

func (s *server) ListStages(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListStagesRequest],
) (*connect.Response[svcv1alpha1.ListStagesResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	var list kargoapi.StageList
	if err := s.client.List(ctx, &list, client.InNamespace(project)); err != nil {
		return nil, fmt.Errorf("list stages: %w", err)
	}

	slices.SortFunc(list.Items, func(a, b kargoapi.Stage) int {
		return strings.Compare(a.Name, b.Name)
	})

	stages := make([]*kargoapi.Stage, len(list.Items))
	for idx := range list.Items {
		stages[idx] = &list.Items[idx]
	}
	return connect.NewResponse(&svcv1alpha1.ListStagesResponse{
		Stages: stages,
	}), nil
}

// @id ListStages
// @Summary List Stages
// @Description List Stage resources from a project's namespace. Returns a
// @Description StageList resource.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Success 200 {object} object "StageList custom resource (github.com/akuity/kargo/api/v1alpha1.StageList)"
// @Router /v1beta1/projects/{project}/stages [get]
func (s *server) listStages(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param("project")

	if watchMode := c.Query("watch") == trueStr; watchMode {
		s.watchStages(c, project)
		return
	}

	list := &kargoapi.StageList{}
	if err := s.client.List(ctx, list, client.InNamespace(project)); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, list)
}

func (s *server) watchStages(c *gin.Context, project string) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	w, err := s.client.Watch(ctx, &kargoapi.StageList{}, client.InNamespace(project))
	if err != nil {
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch stages: %w", err))
		return
	}
	defer w.Stop()

	keepaliveTicker := time.NewTicker(30 * time.Second)
	defer keepaliveTicker.Stop()

	setSSEHeaders(c)

	for {
		select {
		case <-ctx.Done():
			logger.Debug("watch context done", "error", ctx.Err())
			return

		case <-keepaliveTicker.C:
			if !writeSSEKeepalive(c) {
				return
			}

		case e, ok := <-w.ResultChan():
			if !ok {
				logger.Debug("watch channel closed")
				return
			}
			if !convertAndSendWatchEvent(c, e, (*kargoapi.Stage)(nil)) {
				return
			}
		}
	}
}
