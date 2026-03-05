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

func (s *server) GetProjectConfig(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetProjectConfigRequest],
) (*connect.Response[svcv1alpha1.GetProjectConfigResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	// Get the ProjectConfig from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": kargoapi.GroupVersion.String(),
			"kind":       "ProjectConfig",
		},
	}
	if err := s.client.Get(
		ctx, client.ObjectKey{Name: project, Namespace: project}, u,
	); err != nil {
		if client.IgnoreNotFound(err) == nil {
			err = fmt.Errorf("ProjectConfig %q not found", project)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	p, raw, err := objectOrRaw(
		s.client, u, req.Msg.GetFormat(), &kargoapi.ProjectConfig{},
	)
	if err != nil {
		return nil, err
	}
	if raw != nil {
		return connect.NewResponse(&svcv1alpha1.GetProjectConfigResponse{
			Result: &svcv1alpha1.GetProjectConfigResponse_Raw{Raw: raw},
		}), nil
	}
	return connect.NewResponse(&svcv1alpha1.GetProjectConfigResponse{
		Result: &svcv1alpha1.GetProjectConfigResponse_ProjectConfig{
			ProjectConfig: p,
		},
	}), nil
}

// @id GetProjectConfig
// @Summary Retrieve ProjectConfig
// @Description Retrieve the single ProjectConfig resource from a project's
// @Description namespace.
// @Tags Core, Project-Level, Config, Singleton
// @Security BearerAuth
// @Param project path string true "Project name"
// @Produce json
// @Success 200 {object} object "ProjectConfig custom resource (github.com/akuity/kargo/api/v1alpha1.ProjectConfig)"
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
			if !convertAndSendWatchEvent(c, e, (*kargoapi.ProjectConfig)(nil)) {
				return
			}
		}
	}
}
