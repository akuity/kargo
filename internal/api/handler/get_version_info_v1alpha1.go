package handler

import (
	"context"

	"github.com/bufbuild/connect-go"

	typesv1alpha1 "github.com/akuity/kargo/internal/api/types/v1alpha1"
	"github.com/akuity/kargo/internal/version"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

type GetVersionInfoV1Alpha1Func func(
	context.Context,
	*connect.Request[svcv1alpha1.GetVersionInfoRequest],
) (*connect.Response[svcv1alpha1.GetVersionInfoResponse], error)

func GetVersionInfoV1Alpha1(v version.Version) GetVersionInfoV1Alpha1Func {
	return func(
		_ context.Context,
		_ *connect.Request[svcv1alpha1.GetVersionInfoRequest],
	) (*connect.Response[svcv1alpha1.GetVersionInfoResponse], error) {
		return connect.NewResponse(&svcv1alpha1.GetVersionInfoResponse{
			VersionInfo: typesv1alpha1.ToVersionProto(v),
		}), nil
	}
}
