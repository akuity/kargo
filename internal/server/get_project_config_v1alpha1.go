package server

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func (s *server) GetProjectConfig(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetProjectConfigRequest],
) (*connect.Response[svcv1alpha1.GetProjectConfigResponse], error) {
	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	// Get the ProjectConfig from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": kargoapi.GroupVersion.String(),
			"kind":       "ProjectConfig",
		},
	}
	if err := s.client.Get(
		ctx, client.ObjectKey{Name: project, Namespace: project}, u,
	); err != nil {
		if client.IgnoreNotFound(err) == nil {
			err = fmt.Errorf("ProjectConfig %q not found", project)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	p, raw, err := objectOrRaw(
		s.client, u, req.Msg.GetFormat(), &kargoapi.ProjectConfig{},
	)
	if err != nil {
		return nil, err
	}
	if raw != nil {
		return connect.NewResponse(&svcv1alpha1.GetProjectConfigResponse{
			Result: &svcv1alpha1.GetProjectConfigResponse_Raw{Raw: raw},
		}), nil
	}
	return connect.NewResponse(&svcv1alpha1.GetProjectConfigResponse{
		Result: &svcv1alpha1.GetProjectConfigResponse_ProjectConfig{
			ProjectConfig: p,
		},
	}), nil
}
