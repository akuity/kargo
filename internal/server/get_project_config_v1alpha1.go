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
)

func (s *server) GetProjectConfig(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetProjectConfigRequest],
) (*connect.Response[svcv1alpha1.GetProjectConfigResponse], error) {
	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	// Get the ProjectConfig from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": kargoapi.GroupVersion.String(),
			"kind":       "ProjectConfig",
		},
	}
	if err := s.client.Get(ctx, client.ObjectKey{Name: name, Namespace: name}, &u); err != nil {
		if client.IgnoreNotFound(err) == nil {
			err = fmt.Errorf("ProjectConfig %q not found", name)
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
		return connect.NewResponse(&svcv1alpha1.GetProjectConfigResponse{
			Result: &svcv1alpha1.GetProjectConfigResponse_Raw{
				Raw: raw,
			},
		}), nil
	default:
		p := kargoapi.ProjectConfig{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &p); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		obj, _, err := objectOrRaw(&p, req.Msg.GetFormat())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.GetProjectConfigResponse{
			Result: &svcv1alpha1.GetProjectConfigResponse_ProjectConfig{
				ProjectConfig: obj,
			},
		}), nil
	}
}
