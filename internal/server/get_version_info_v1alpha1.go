package server

import (
	"context"

	"connectrpc.com/connect"

	"github.com/akuity/kargo/internal/version"
	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) GetVersionInfo(
	context.Context,
	*connect.Request[svcv1alpha1.GetVersionInfoRequest],
) (*connect.Response[svcv1alpha1.GetVersionInfoResponse], error) {
	return connect.NewResponse(
		&svcv1alpha1.GetVersionInfoResponse{
			VersionInfo: svcv1alpha1.ToVersionProto(version.GetVersion()),
		},
	), nil
}
