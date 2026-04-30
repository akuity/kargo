package server

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
)

func (s *server) DeleteClusterConfig(
	ctx context.Context,
	_ *connect.Request[svcv1alpha1.DeleteClusterConfigRequest],
) (*connect.Response[svcv1alpha1.DeleteClusterConfigResponse], error) {
	if err := s.client.Delete(
		ctx,
		&kargoapi.ClusterConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: api.ClusterConfigName,
			},
		},
	); err != nil {
		return nil, fmt.Errorf("delete ClusterConfig: %w", err)
	}
	return connect.NewResponse(&svcv1alpha1.DeleteClusterConfigResponse{}), nil
}

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
