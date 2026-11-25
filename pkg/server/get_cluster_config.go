package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
	"github.com/akuity/kargo/pkg/logging"
)

// @id GetClusterConfig
// @Summary Retrieve the ClusterConfig
// @Description Retrieve the single ClusterConfig resource.
// @Tags System, Config, Cluster-Scoped Resource, Singleton
// @Security BearerAuth
// @Produce json
// @Success 200 {object} kargoapi.ClusterConfig "ClusterConfig custom resource"
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

	watcher, err := s.watchK8sResources[*kargoapi.ClusterConfig](
		ctx,
		&kargoapi.ClusterConfigList{},
		client.MatchingFields{"metadata.name": api.ClusterConfigName},
	)
	if err != nil {
		if SendSSEWatchStartError(c, err) {
			return
		}
		logger.Error(err, "failed to start watch")
		_ = c.Error(fmt.Errorf("watch cluster config: %w", err))
		return
	}
	s.watchResources(c, watcher, nil)
}
