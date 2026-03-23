package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

func (s *server) GetStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetStageRequest],
) (*connect.Response[svcv1alpha1.GetStageResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	// Get the Stage from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": kargoapi.GroupVersion.String(),
			"kind":       "Stage",
		},
	}
	if err := s.client.Get(
		ctx, client.ObjectKey{Name: name, Namespace: project}, u,
	); err != nil {
		if client.IgnoreNotFound(err) == nil {
			// nolint:staticcheck
			err = fmt.Errorf("Stage %q not found in project %q", name, project)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	stage, raw, err := objectOrRaw(
		s.client, u, req.Msg.GetFormat(), &kargoapi.Stage{},
	)
	if err != nil {
		return nil, err
	}
	if raw != nil {
		return connect.NewResponse(&svcv1alpha1.GetStageResponse{
			Result: &svcv1alpha1.GetStageResponse_Raw{Raw: raw},
		}), nil
	}
	return connect.NewResponse(&svcv1alpha1.GetStageResponse{
		Result: &svcv1alpha1.GetStageResponse_Stage{Stage: stage},
	}), nil
}

// @id GetStage
// @Summary Retrieve a Stage
// @Description Retrieve a Stage resource from a project's namespace.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Produce json
// @Param project path string true "Project name"
// @Param stage path string true "Stage name"
// @Success 200 {object} object "Stage custom resource (github.com/akuity/kargo/api/v1alpha1.Stage)"
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
