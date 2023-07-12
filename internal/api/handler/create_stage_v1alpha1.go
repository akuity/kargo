package handler

import (
	"context"

	"github.com/bufbuild/connect-go"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type CreateStageV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.CreateStageRequest],
) (*connect.Response[svcv1alpha1.CreateStageResponse], error)

func CreateStageV1Alpha1(
	kc client.Client,
) CreateStageV1Alpha1Func {
	return func(
		ctx context.Context,
		req *connect.Request[svcv1alpha1.CreateStageRequest],
	) (*connect.Response[svcv1alpha1.CreateStageResponse], error) {
		var stage kubev1alpha1.Stage
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
			stage = kubev1alpha1.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: req.Msg.GetTyped().GetProject(),
					Name:      req.Msg.GetTyped().GetName(),
				},
				Spec: fromStageSpecProto(req.Msg.GetTyped().GetSpec()),
			}
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("environment should not be empty"))
		}
		if err := kc.Create(ctx, &stage); err != nil {
			if kubeerr.IsNotFound(err) {
				return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", stage.GetNamespace()))
			}
			if kubeerr.IsAlreadyExists(err) {
				return nil, connect.NewError(connect.CodeAlreadyExists, err)
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.CreateStageResponse{
			Stage: toStageProto(stage),
		}), nil
	}
}
