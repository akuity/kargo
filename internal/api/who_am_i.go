package api

import (
	"context"

	"connectrpc.com/connect"

	svcv1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
)

func (s *server) WhoAmI(
	_ context.Context,
	_ *connect.Request[svcv1alpha1.WhoAmIRequest],
) (*connect.Response[svcv1alpha1.WhoAmIResponse], error) {
	// request that UI uses for token validity
	// this can be extended to re-use in CLI
	return &connect.Response[svcv1alpha1.WhoAmIResponse]{}, nil
}
