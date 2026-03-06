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

func (s *server) GetPromotion(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetPromotionRequest],
) (*connect.Response[svcv1alpha1.GetPromotionResponse], error) {
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

	// Get the Promotion from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": kargoapi.GroupVersion.String(),
			"kind":       "Promotion",
		},
	}
	if err := s.client.Get(
		ctx, client.ObjectKey{Name: name, Namespace: project}, u,
	); err != nil {
		if client.IgnoreNotFound(err) == nil {
			// nolint:staticcheck
			err = fmt.Errorf("Promotion %q not found in project %q", name, project)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	p, raw, err := objectOrRaw(
		s.client, u, req.Msg.GetFormat(), &kargoapi.Promotion{},
	)
	if err != nil {
		return nil, err
	}
	if raw != nil {
		return connect.NewResponse(&svcv1alpha1.GetPromotionResponse{
			Result: &svcv1alpha1.GetPromotionResponse_Raw{Raw: raw},
		}), nil
	}
	return connect.NewResponse(&svcv1alpha1.GetPromotionResponse{
		Result: &svcv1alpha1.GetPromotionResponse_Promotion{Promotion: p},
	}), nil
}

// @id GetPromotion
// @Summary Retrieve a Promotion
// @Description Retrieve a Promotion resource from a project's namespace.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param promotion path string true "Promotion name"
// @Produce json
// @Success 200 {object} object "Promotion custom resource (github.com/akuity/kargo/api/v1alpha1.Promotion)"
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
			if !convertAndSendWatchEvent(c, e, (*kargoapi.Promotion)(nil)) {
				return
			}
		}
	}
}
