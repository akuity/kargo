package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

// @id DeleteClusterConfig
// @Summary Delete the ClusterConfig
// @Description Deletes the single ClusterConfig resource.
// @Tags System, Config, Cluster-Scoped Resource, Singleton
// @Security BearerAuth
// @Success 204 "Deleted successfully"
// @Router /v1beta1/system/cluster-config [delete]
func (s *server) deleteClusterConfig(c *gin.Context) {
	ctx := c.Request.Context()

	if err := s.client.Delete(
		ctx,
		&kargoapi.ClusterConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: api.ClusterConfigName,
			},
		},
	); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
