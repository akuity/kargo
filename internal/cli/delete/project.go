package delete

import (
	"context"

	"connectrpc.com/connect"

	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

func deleteProject(
	ctx context.Context,
	kargoSvcCli svcv1alpha1connect.KargoServiceClient,
	name string,
) error {
	_, err := kargoSvcCli.DeleteProject(ctx, connect.NewRequest(&v1alpha1.DeleteProjectRequest{
		Name: name,
	}))
	return err
}
