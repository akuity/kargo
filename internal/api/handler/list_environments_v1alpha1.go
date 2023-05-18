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
	"github.com/akuity/kargo/pkg/api/v1alpha1"
)

type ListEnvironmentsV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.ListEnvironmentsRequest],
) (*connect.Response[svcv1alpha1.ListEnvironmentsResponse], error)

func ListEnvironmentsV1Alpha1(
	kc client.Client,
) ListEnvironmentsV1Alpha1Func {
	return func(
		ctx context.Context,
		req *connect.Request[svcv1alpha1.ListEnvironmentsRequest],
	) (*connect.Response[svcv1alpha1.ListEnvironmentsResponse], error) {
		if req.Msg.GetNamespace() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("namespace should not be empty"))
		}

		if err := kc.Get(ctx, client.ObjectKey{Name: req.Msg.GetNamespace()}, &corev1.Namespace{}); err != nil {
			if kubeerr.IsNotFound(err) {
				return nil, connect.NewError(connect.CodeNotFound,
					fmt.Errorf("namespace %q not found", req.Msg.GetNamespace()))
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		var list kubev1alpha1.EnvironmentList
		if err := kc.List(ctx, &list, client.InNamespace(req.Msg.GetNamespace())); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		envs := make([]*v1alpha1.Environment, len(list.Items))
		for idx := range list.Items {
			envs[idx] = toEnvironmentProto(list.Items[idx])
		}
		return connect.NewResponse(&svcv1alpha1.ListEnvironmentsResponse{
			Environments: envs,
		}), nil
	}
}
