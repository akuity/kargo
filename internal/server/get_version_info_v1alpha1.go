package server

import (
	"context"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/x/version"
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
