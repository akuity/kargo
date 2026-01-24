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
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/logging"
)

func (s *server) GetClusterConfig(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetClusterConfigRequest],
) (*connect.Response[svcv1alpha1.GetClusterConfigResponse], error) {
	// Get the ClusterConfig from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": kargoapi.GroupVersion.String(),
			"kind":       "ClusterConfig",
		},
	}
	if err := s.client.Get(
		ctx, client.ObjectKey{Name: api.ClusterConfigName}, u,
	); err != nil {
		if client.IgnoreNotFound(err) == nil {
			err = fmt.Errorf("ClusterConfig %q not found", api.ClusterConfigName)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	cfg, raw, err := objectOrRaw(
		s.client, u, req.Msg.GetFormat(), &kargoapi.ClusterConfig{},
	)
	if err != nil {
		return nil, err
	}
	if raw != nil {
		return connect.NewResponse(&svcv1alpha1.GetClusterConfigResponse{
			Result: &svcv1alpha1.GetClusterConfigResponse_Raw{Raw: raw},
		}), nil
	}
	return connect.NewResponse(&svcv1alpha1.GetClusterConfigResponse{
		Result: &svcv1alpha1.GetClusterConfigResponse_ClusterConfig{
			ClusterConfig: cfg,
		},
	}), nil
}

// @id GetClusterConfig
// @Summary Retrieve the ClusterConfig
// @Description Retrieve the single ClusterConfig resource.
// @Tags System, Config, Cluster-Scoped Resource, Singleton
// @Security BearerAuth
// @Produce json
// @Success 200 {object} object "ClusterConfig custom resource (github.com/akuity/kargo/api/v1alpha1.ClusterConfig)"
// @Router /v1beta1/system/cluster-config [get]
func (s *server) getClusterConfig(c *gin.Context) {
	ctx := c.Request.Context()

	if watchMode := c.Query("watch") == trueStr; watchMode {
		s.watchClusterConfig(c)
		return
	}

	config := &kargoapi.ClusterConfig{}
	if err := s.client.Get(
		ctx, client.ObjectKey{Name: api.ClusterConfigName}, config,
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, config)
}

func (s *server) watchClusterConfig(c *gin.Context) {
	ctx := c.Request.Context()
	logger := logging.LoggerFromContext(ctx)

	// Validate that the ClusterConfig exists before starting the watch
	config := &kargoapi.ClusterConfig{}
	if err := s.client.Get(
		ctx,
		client.ObjectKey{Name: api.ClusterConfigName},
		config,
	); err != nil {
		_ = c.Error(err)
		return
	}

	// ClusterConfig is cluster-scoped, so no namespace
	w, err := s.client.Watch(
		ctx,
		&kargoapi.ClusterConfigList{},
		client.MatchingFields{"metadata.name": api.ClusterConfigName},
	)
	if err != nil {
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch cluster config: %w", err))
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
			if !convertAndSendWatchEvent(c, e, (*kargoapi.ClusterConfig)(nil)) {
				return
			}
		}
	}
}
