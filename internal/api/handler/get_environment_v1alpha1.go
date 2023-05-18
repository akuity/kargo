package handler

import (
	"context"
	"fmt"

	"github.com/bufbuild/connect-go"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubev1alpha1 "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type GetEnvironmentV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.GetEnvironmentRequest],
) (*connect.Response[svcv1alpha1.GetEnvironmentResponse], error)

func GetEnvironmentV1Alpha1(
	kc client.Client,
) GetEnvironmentV1Alpha1Func {
	return func(
		ctx context.Context,
		req *connect.Request[svcv1alpha1.GetEnvironmentRequest],
	) (*connect.Response[svcv1alpha1.GetEnvironmentResponse], error) {
		if req.Msg.GetNamespace() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("namespace should not be empty"))
		}
		if req.Msg.GetName() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name should not be empty"))
		}

		if err := kc.Get(ctx, client.ObjectKey{Name: req.Msg.GetNamespace()}, &corev1.Namespace{}); err != nil {
			if kubeerr.IsNotFound(err) {
				return nil, connect.NewError(connect.CodeNotFound,
					fmt.Errorf("namespace %q not found", req.Msg.GetNamespace()))
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		var env kubev1alpha1.Environment
		if err := kc.Get(ctx, client.ObjectKey{
			Namespace: req.Msg.GetNamespace(),
			Name:      req.Msg.GetName(),
		}, &env); err != nil {
			if kubeerr.IsNotFound(err) {
				return nil, connect.NewError(connect.CodeNotFound, err)
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.GetEnvironmentResponse{
			Environment: toEnvironmentProto(env),
		}), nil
	}
}
