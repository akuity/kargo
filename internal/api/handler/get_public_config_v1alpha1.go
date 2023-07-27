package handler

import (
	"context"

	"github.com/bufbuild/connect-go"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type GetPublicConfigV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.GetPublicConfigRequest],
) (*connect.Response[svcv1alpha1.GetPublicConfigResponse], error)

func GetPublicConfigV1Alpha1(
	cfg *svcv1alpha1.GetPublicConfigResponse,
) GetPublicConfigV1Alpha1Func {
	return func(context.Context, *connect.Request[svcv1alpha1.GetPublicConfigRequest],
	) (*connect.Response[svcv1alpha1.GetPublicConfigResponse], error) {
		return connect.NewResponse(cfg), nil
	}
}
