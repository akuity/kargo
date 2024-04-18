package api

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) GetStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.GetStageRequest],
) (*connect.Response[svcv1alpha1.GetStageResponse], error) {
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

	var stage kargoapi.Stage
	if err := s.client.Get(ctx, client.ObjectKey{
		Namespace: project,
		Name:      name,
	}, &stage); err != nil {
		return nil, fmt.Errorf("get stage: %w", err)
	}

	obj, raw, err := objectOrRaw(&stage, req.Msg.GetFormat())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if raw != nil {
		return connect.NewResponse(&svcv1alpha1.GetStageResponse{
			Result: &svcv1alpha1.GetStageResponse_Raw{
				Raw: raw,
			},
		}), nil
	}
	return connect.NewResponse(&svcv1alpha1.GetStageResponse{
		Result: &svcv1alpha1.GetStageResponse_Stage{
			Stage: obj,
		},
	}), nil
}
