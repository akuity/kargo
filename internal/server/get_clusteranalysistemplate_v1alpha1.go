package server

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) GetClusterAnalysisTemplate(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetClusterAnalysisTemplateRequest],
) (*connect.Response[svcv1alpha1.GetClusterAnalysisTemplateResponse], error) {
	if !s.cfg.RolloutsIntegrationEnabled {
		return nil, connect.NewError(
			connect.CodeUnimplemented,
			errors.New("Argo Rollouts integration is not enabled"),
		)
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	// Get the ClusterAnalysisTemplate from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": rolloutsapi.GroupVersion.String(),
			"kind":       "ClusterAnalysisTemplate",
		},
	}
	if err := s.client.Get(ctx, types.NamespacedName{
		Name: name,
	}, &u); err != nil {
		if client.IgnoreNotFound(err) == nil {
			err = fmt.Errorf("ClusterAnalysisTemplate %q not found", name)
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
		return connect.NewResponse(&svcv1alpha1.GetClusterAnalysisTemplateResponse{
			Result: &svcv1alpha1.GetClusterAnalysisTemplateResponse_Raw{
				Raw: raw,
			},
		}), nil
	default:
		at := rolloutsapi.ClusterAnalysisTemplate{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &at); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		obj, _, err := objectOrRaw(&at, req.Msg.GetFormat())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.GetClusterAnalysisTemplateResponse{
			Result: &svcv1alpha1.GetClusterAnalysisTemplateResponse_ClusterAnalysisTemplate{
				ClusterAnalysisTemplate: obj,
			},
		}), nil
	}
}
