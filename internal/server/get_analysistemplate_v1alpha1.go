package server

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	rolloutsapi "github.com/akuity/kargo/api/stubs/rollouts/v1alpha1"
)

func (s *server) GetAnalysisTemplate(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetAnalysisTemplateRequest],
) (*connect.Response[svcv1alpha1.GetAnalysisTemplateResponse], error) {
	if !s.cfg.RolloutsIntegrationEnabled {
		// nolint:staticcheck
		return nil, connect.NewError(
			connect.CodeUnimplemented,
			errors.New("Argo Rollouts integration is not enabled"),
		)
	}

	project := req.Msg.GetProject()
	if err := validateFieldNotEmpty("project", project); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	if err := s.validateProjectExists(ctx, project); err != nil {
		return nil, err
	}

	// Get the AnalysisTemplate from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": rolloutsapi.GroupVersion.String(),
			"kind":       "AnalysisTemplate",
		},
	}
	if err := s.client.Get(
		ctx, client.ObjectKey{Namespace: project, Name: name}, u,
	); err != nil {
		if client.IgnoreNotFound(err) == nil {
			err = fmt.Errorf("AnalysisTemplate %q not found in namespace %q", name, project)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	at, raw, err := objectOrRaw(
		s.client, u, req.Msg.GetFormat(), &rolloutsapi.AnalysisTemplate{},
	)
	if err != nil {
		return nil, err
	}
	if raw != nil {
		return connect.NewResponse(&svcv1alpha1.GetAnalysisTemplateResponse{
			Result: &svcv1alpha1.GetAnalysisTemplateResponse_Raw{Raw: raw},
		}), nil
	}
	return connect.NewResponse(&svcv1alpha1.GetAnalysisTemplateResponse{
		Result: &svcv1alpha1.GetAnalysisTemplateResponse_AnalysisTemplate{
			AnalysisTemplate: at,
		},
	}), nil
}
