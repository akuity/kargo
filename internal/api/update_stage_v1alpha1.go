package api

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) UpdateStage(
	ctx context.Context,
	req *connect.Request[svcv1alpha1.UpdateStageRequest],
) (*connect.Response[svcv1alpha1.UpdateStageResponse], error) {
	var stage kargoapi.Stage
	switch {
	case req.Msg.GetYaml() != "":
		if err := yaml.Unmarshal([]byte(req.Msg.GetYaml()), &stage); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrap(err, "invalid yaml"))
		}
	case req.Msg.GetTyped() != nil:
		if req.Msg.GetTyped().GetProject() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
		}
		if req.Msg.GetTyped().GetName() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name should not be empty"))
		}
		stage = kargoapi.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Msg.GetTyped().GetProject(),
				Name:      req.Msg.GetTyped().GetName(),
			},
			Spec: typesv1alpha1.FromStageSpecProto(req.Msg.GetTyped().GetSpec()),
		}
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("stage should not be empty"))
	}

	if err := s.validateProject(ctx, stage.GetNamespace()); err != nil {
		return nil, err
	}
	var existingStage kargoapi.Stage
	if err := s.client.Get(ctx, client.ObjectKeyFromObject(&stage), &existingStage); err != nil {
		if kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeUnknown, errors.Wrap(err, "get stage"))
	}
	stage.SetResourceVersion(existingStage.GetResourceVersion())
	if err := s.client.Update(ctx, &stage); err != nil {
		return nil, connect.NewError(connect.CodeUnknown, errors.Wrap(err, "update stage"))
	}
	return connect.NewResponse(&svcv1alpha1.UpdateStageResponse{
		Stage: typesv1alpha1.ToStageProto(stage),
	}), nil
}
