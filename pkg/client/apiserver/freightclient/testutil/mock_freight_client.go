package testutil

import (
	"context"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api/freightapi"
	"github.com/akuity/kargo/pkg/client/apiserver/freightclient"
)

var _ freightclient.FreightClient = &MockFreightClient{}

type MockFreightClient struct {
	GetFreightFun    func(ctx context.Context, project, name string) (*kargoapi.Freight, error)
	CreateFreightFun func(
		ctx context.Context,
		project string,
		freight *kargoapi.Freight,
		metadata map[string]apiextensionsv1.JSON,
	) (*kargoapi.Freight, bool, error)
	DeleteFreightFun        func(ctx context.Context, project, nameOrAlias string) error
	PatchFreightAliasFun    func(ctx context.Context, project, nameOrAlias, newAlias string) error
	PatchFreightMetadataFun func(
		ctx context.Context,
		project, freightName string,
		metadata map[string]any,
	) error
	ListFreightFromOriginFun func(
		ctx context.Context,
		project string,
		origin kargoapi.FreightOrigin,
		opts *freightapi.ListFreightFromOriginOptions,
	) ([]kargoapi.Freight, error)
	MarkFreightsCurrentlyInStageFun func(
		ctx context.Context,
		project, stage string,
		freights []string,
	) error
	MarkFreightsVerifiedInStageFun func(
		ctx context.Context,
		project, stage string,
		freights []string,
		verifiedAt time.Time,
	) error
	ClearStageFreightsFun func(ctx context.Context, project, stage string) error
}

func (mock *MockFreightClient) GetFreight(ctx context.Context, project, name string) (*kargoapi.Freight, error) {
	return mock.GetFreightFun(ctx, project, name)
}
func (mock *MockFreightClient) CreateFreight(
	ctx context.Context,
	project string,
	freight *kargoapi.Freight,
	metadata map[string]apiextensionsv1.JSON,
) (*kargoapi.Freight, bool, error) {
	return mock.CreateFreightFun(ctx, project, freight, metadata)
}
func (mock *MockFreightClient) DeleteFreight(ctx context.Context, project, nameOrAlias string) error {
	return mock.DeleteFreightFun(ctx, project, nameOrAlias)
}
func (mock *MockFreightClient) PatchFreightAlias(ctx context.Context, project, nameOrAlias, newAlias string) error {
	return mock.PatchFreightAliasFun(ctx, project, nameOrAlias, newAlias)
}
func (mock *MockFreightClient) PatchFreightMetadata(
	ctx context.Context,
	project, freightName string,
	metadata map[string]any,
) error {
	return mock.PatchFreightMetadataFun(ctx, project, freightName, metadata)
}
func (mock *MockFreightClient) ListFreightFromOrigin(
	ctx context.Context,
	project string,
	origin kargoapi.FreightOrigin,
	opts *freightapi.ListFreightFromOriginOptions,
) ([]kargoapi.Freight, error) {
	return mock.ListFreightFromOriginFun(ctx, project, origin, opts)
}
func (mock *MockFreightClient) MarkFreightsCurrentlyInStage(
	ctx context.Context,
	project, stage string,
	freights []string,
) error {
	return mock.MarkFreightsCurrentlyInStageFun(ctx, project, stage, freights)
}
func (mock *MockFreightClient) MarkFreightsVerifiedInStage(
	ctx context.Context,
	project, stage string,
	freights []string,
	verifiedAt time.Time,
) error {
	return mock.MarkFreightsVerifiedInStageFun(ctx, project, stage, freights, verifiedAt)
}
func (mock *MockFreightClient) ClearStageFreights(ctx context.Context, project, stage string) error {
	return mock.ClearStageFreightsFun(ctx, project, stage)
}
