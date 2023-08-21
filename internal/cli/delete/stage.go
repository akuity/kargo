package delete

import (
	"context"

	"connectrpc.com/connect"

	v1alpha1 "github.com/akuity/kargo/pkg/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/api/service/v1alpha1/svcv1alpha1connect"
)

func deleteStage(
	ctx context.Context,
	kargoSvcCli svcv1alpha1connect.KargoServiceClient,
	project string,
	name string,
) error {
	_, err := kargoSvcCli.DeleteStage(ctx, connect.NewRequest(&v1alpha1.DeleteStageRequest{
		Project: project,
		Name:    name,
	}))
	return err
}
