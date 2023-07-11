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

type ListStagesV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.ListStagesRequest],
) (*connect.Response[svcv1alpha1.ListStagesResponse], error)

func ListStagesV1Alpha1(
	kc client.Client,
) ListStagesV1Alpha1Func {
	return func(
		ctx context.Context,
		req *connect.Request[svcv1alpha1.ListStagesRequest],
	) (*connect.Response[svcv1alpha1.ListStagesResponse], error) {
		if req.Msg.GetProject() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
		}

		if err := kc.Get(ctx, client.ObjectKey{Name: req.Msg.GetProject()}, &corev1.Namespace{}); err != nil {
			if kubeerr.IsNotFound(err) {
				return nil, connect.NewError(connect.CodeNotFound,
					fmt.Errorf("project %q not found", req.Msg.GetProject()))
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		var list kubev1alpha1.StageList
		if err := kc.List(ctx, &list, client.InNamespace(req.Msg.GetProject())); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		stages := make([]*v1alpha1.Stage, len(list.Items))
		for idx := range list.Items {
			stages[idx] = toStageProto(list.Items[idx])
		}
		return connect.NewResponse(&svcv1alpha1.ListStagesResponse{
			Stages: stages,
		}), nil
	}
}
