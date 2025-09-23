package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api"
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
