package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/api"
)

func (s *server) GetClusterConfig(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetClusterConfigRequest],
) (*connect.Response[svcv1alpha1.GetClusterConfigResponse], error) {
	// Get the ClusterConfig from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": kargoapi.GroupVersion.String(),
			"kind":       "ClusterConfig",
		},
	}
	if err := s.client.Get(ctx, client.ObjectKey{Name: api.ClusterConfigName}, &u); err != nil {
		if client.IgnoreNotFound(err) == nil {
			err = fmt.Errorf("ClusterConfig %q not found", api.ClusterConfigName)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	switch req.Msg.GetFormat() {
	case svcv1alpha1.RawFormat_RAW_FORMAT_JSON, svcv1alpha1.RawFormat_RAW_FORMAT_YAML:
		_, raw, err := objectOrRaw(&u, req.Msg.GetFormat())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.GetClusterConfigResponse{
			Result: &svcv1alpha1.GetClusterConfigResponse_Raw{
				Raw: raw,
			},
		}), nil
	default:
		p := kargoapi.ClusterConfig{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &p); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		obj, _, err := objectOrRaw(&p, req.Msg.GetFormat())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.GetClusterConfigResponse{
			Result: &svcv1alpha1.GetClusterConfigResponse_ClusterConfig{
				ClusterConfig: obj,
			},
		}), nil
	}
}
