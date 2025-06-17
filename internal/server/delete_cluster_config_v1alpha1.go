package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
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
