package server

import (
	"connectrpc.com/grpchealth"

	"github.com/akuity/kargo/api/service/v1alpha1/svcv1alpha1connect"
)

func NewHealthChecker() grpchealth.Checker {
	return grpchealth.NewStaticChecker(
		svcv1alpha1connect.KargoServiceName,
	)
}
