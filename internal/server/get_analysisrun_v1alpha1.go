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

func (s *server) GetAnalysisRun(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetAnalysisRunRequest],
) (*connect.Response[svcv1alpha1.GetAnalysisRunResponse], error) {
	if !s.cfg.RolloutsIntegrationEnabled {
		// nolint:staticcheck
		return nil, connect.NewError(
			connect.CodeUnimplemented,
			errors.New("Argo Rollouts integration is not enabled"),
		)
	}

	namespace := req.Msg.GetNamespace()
	if err := validateFieldNotEmpty("namespace", namespace); err != nil {
		return nil, err
	}

	name := req.Msg.GetName()
	if err := validateFieldNotEmpty("name", name); err != nil {
		return nil, err
	}

	// Get the AnalysisRun from the Kubernetes API as an unstructured object.
	// Using an unstructured object allows us to return the object _as presented
	// by the API_ if a raw format is requested.
	u := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": rolloutsapi.GroupVersion.String(),
			"kind":       "AnalysisRun",
		},
	}
	if err := s.client.Get(
		ctx, client.ObjectKey{Namespace: namespace, Name: name}, u,
	); err != nil {
		if client.IgnoreNotFound(err) == nil {
			err = fmt.Errorf("AnalysisRun %q not found in namespace %q", name, namespace)
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	ar, raw, err := objectOrRaw(
		s.client, u, req.Msg.GetFormat(), &rolloutsapi.AnalysisRun{},
	)
	if err != nil {
		return nil, err
	}
	if raw != nil {
		return connect.NewResponse(&svcv1alpha1.GetAnalysisRunResponse{
			Result: &svcv1alpha1.GetAnalysisRunResponse_Raw{Raw: raw},
		}), nil
	}
	return connect.NewResponse(&svcv1alpha1.GetAnalysisRunResponse{
		Result: &svcv1alpha1.GetAnalysisRunResponse_AnalysisRun{AnalysisRun: ar},
	}), nil
}
