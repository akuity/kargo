package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

// @id RefreshClusterConfig
// @Summary Refresh the ClusterConfig
// @Description Refresh the single ClusterConfig resource. Refreshing enqueues
// @Description the resource for reconciliation by its corresponding controller.
// @Tags System, Config, Cluster-Scoped Resource, Singleton
// @Security BearerAuth
// @Produce json
// @Success 200 "Success"
// @Router /v1beta1/system/cluster-config/refresh [post]
func (s *server) refreshClusterConfig(c *gin.Context) {
	ctx := c.Request.Context()

	obj := &kargoapi.ClusterConfig{
		ObjectMeta: v1.ObjectMeta{Name: api.ClusterConfigName},
	}

	if err := api.RefreshObject(ctx, s.client.InternalClient(), obj); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}
