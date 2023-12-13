package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/v1alpha1"
)

func (s *server) ListStages(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.ListStagesRequest],
) (*connect.Response[svcv1alpha1.ListStagesResponse], error) {
	if req.Msg.GetProject() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
	}
	if err := s.validateProject(ctx, req.Msg.GetProject()); err != nil {
		return nil, err
	}

	var list kargoapi.StageList
	if err := s.client.List(ctx, &list, client.InNamespace(req.Msg.GetProject())); err != nil {
		return nil, connect.NewError(getCodeFromError(err), errors.Wrap(err, "list stages"))
	}

	stages := make([]*v1alpha1.Stage, len(list.Items))
	for idx := range list.Items {
		stages[idx] = typesv1alpha1.ToStageProto(list.Items[idx])
	}
	return connect.NewResponse(&svcv1alpha1.ListStagesResponse{
		Stages: stages,
	}), nil
}
