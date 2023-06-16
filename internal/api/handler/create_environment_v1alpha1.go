package handler

import (
	"context"

	"github.com/bufbuild/connect-go"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type CreateEnvironmentV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.CreateEnvironmentRequest],
) (*connect.Response[svcv1alpha1.CreateEnvironmentResponse], error)

func CreateEnvironmentV1Alpha1(
	kc client.Client,
) CreateEnvironmentV1Alpha1Func {
	return func(
		ctx context.Context,
		req *connect.Request[svcv1alpha1.CreateEnvironmentRequest],
	) (*connect.Response[svcv1alpha1.CreateEnvironmentResponse], error) {
		if req.Msg.GetProject() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
		}
		if req.Msg.GetName() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name should not be empty"))
		}

		env := kubev1alpha1.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: req.Msg.GetProject(),
				Name:      req.Msg.GetName(),
			},
			Spec: fromEnvironmentSpecProto(req.Msg.GetSpec()),
		}
		if err := kc.Create(ctx, &env); err != nil {
			if kubeerr.IsAlreadyExists(err) {
				return nil, connect.NewError(connect.CodeAlreadyExists, err)
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.CreateEnvironmentResponse{
			Environment: toEnvironmentProto(env),
		}), nil
	}
}
