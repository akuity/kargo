package handler

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	kubeerr "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type DeleteStageV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.DeleteStageRequest],
) (*connect.Response[svcv1alpha1.DeleteStageResponse], error)

func DeleteStageV1Alpha1(
	kc client.Client,
) DeleteStageV1Alpha1Func {
	validateProject := newProjectValidator(kc)
	return func(
		ctx context.Context,
		req *connect.Request[svcv1alpha1.DeleteStageRequest],
	) (*connect.Response[svcv1alpha1.DeleteStageResponse], error) {
		if req.Msg.GetProject() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("project should not be empty"))
		}
		if req.Msg.GetName() == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name should not be empty"))
		}
		if err := validateProject(ctx, req.Msg.GetProject()); err != nil {
			return nil, err
		}

		var stage kargoapi.Stage
		key := client.ObjectKey{
			Namespace: req.Msg.GetProject(),
			Name:      req.Msg.GetName(),
		}
		if err := kc.Get(ctx, key, &stage); err != nil {
			if kubeerr.IsNotFound(err) {
				return nil, connect.NewError(connect.CodeNotFound,
					fmt.Errorf("stage %q not found", key.String()))
			}
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if err := kc.Delete(ctx, &stage); err != nil && !kubeerr.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&svcv1alpha1.DeleteStageResponse{}), nil
	}
}
