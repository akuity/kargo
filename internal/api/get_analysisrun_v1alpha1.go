package api

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"k8s.io/apimachinery/pkg/types"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) GetAnalysisRun(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetAnalysisRunRequest],
) (*connect.Response[svcv1alpha1.GetAnalysisRunResponse], error) {
	if !s.cfg.RolloutsIntegrationEnabled {
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

	ar, err := s.getAnalysisRunFn(ctx, s.client, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	})
	if err != nil {
		return nil, err
	}
	if ar == nil {
		err = fmt.Errorf("AnalysisRun %q not found in namespace %q", name, namespace)
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	obj, raw, err := objectOrRaw(ar, req.Msg.GetFormat())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if raw != nil {
		return connect.NewResponse(&svcv1alpha1.GetAnalysisRunResponse{
			Result: &svcv1alpha1.GetAnalysisRunResponse_Raw{
				Raw: raw,
			},
		}), nil
	}
	return connect.NewResponse(&svcv1alpha1.GetAnalysisRunResponse{
		Result: &svcv1alpha1.GetAnalysisRunResponse_AnalysisRun{
			AnalysisRun: obj,
		},
	}), nil
}
