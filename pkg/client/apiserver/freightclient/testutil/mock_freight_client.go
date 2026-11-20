package testutil

import (
	"context"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api/freightapi"
)

type MockFreightClient struct {
	CreateFreightFun func(
		ctx context.Context,
		project string,
		freight *kargoapi.Freight,
		metadata map[string]apiextensionsv1.JSON,
	) (*kargoapi.Freight, bool, error)
	ListFreightFromWarehouseFun func(
		ctx context.Context,
		project, warehouse string,
		opts *freightapi.ListWarehouseFreightOptions,
	) ([]kargoapi.Freight, error)
	MarkFreightCurrentlyInStageFun func(
		ctx context.Context,
		project, freight string,
		origin kargoapi.FreightOrigin,
		stage string,
	) error
	MarkFreightVerifiedInStageFun func(
		ctx context.Context,
		project, freight, stage string,
		verifiedAt time.Time,
	) error
	ClearFreightStageStatusFun   func(ctx context.Context, project, freight, stage string) error
	ListFreightByCurrentStageFun func(
		ctx context.Context,
		project, stage string,
	) ([]kargoapi.Freight, error)
	GetFreightFun           func(ctx context.Context, project, name string) (*kargoapi.Freight, error)
	DeleteFreightFun        func(ctx context.Context, project, nameOrAlias string) error
	PatchFreightAliasFun    func(ctx context.Context, project, nameOrAlias, newAlias string) error
	PatchFreightMetadataFun func(
		ctx context.Context,
		project, freightName string,
		metadata map[string]any,
	) error
	ListFreightReferencingStageFun func(
		ctx context.Context,
		project, stage string,
	) ([]kargoapi.Freight, error)
}

func (mock *MockFreightClient) CreateFreight(
	ctx context.Context,
	project string,
	freight *kargoapi.Freight,
	metadata map[string]apiextensionsv1.JSON,
) (*kargoapi.Freight, bool, error) {
	return mock.CreateFreightFun(ctx, project, freight, metadata)
}
func (mock *MockFreightClient) ListFreightFromWarehouse(
	ctx context.Context,
	project, warehouse string,
	opts *freightapi.ListWarehouseFreightOptions,
) ([]kargoapi.Freight, error) {
	return mock.ListFreightFromWarehouseFun(ctx, project, warehouse, opts)
}
func (mock *MockFreightClient) MarkFreightCurrentlyInStage(
	ctx context.Context,
	project, freight string,
	origin kargoapi.FreightOrigin,
	stage string,
) error {
	return mock.MarkFreightCurrentlyInStageFun(ctx, project, freight, origin, stage)
}
func (mock *MockFreightClient) MarkFreightVerifiedInStage(
	ctx context.Context,
	project, freight, stage string,
	verifiedAt time.Time,
) error {
	return mock.MarkFreightVerifiedInStageFun(ctx, project, freight, stage, verifiedAt)
}
func (mock *MockFreightClient) ClearFreightStageStatus(ctx context.Context, project, freight, stage string) error {
	return mock.ClearFreightStageStatusFun(ctx, project, freight, stage)
}
func (mock *MockFreightClient) ListFreightByCurrentStage(
	ctx context.Context,
	project, stage string,
) ([]kargoapi.Freight, error) {
	return mock.ListFreightByCurrentStageFun(ctx, project, stage)
}
func (mock *MockFreightClient) GetFreight(ctx context.Context, project, name string) (*kargoapi.Freight, error) {
	return mock.GetFreightFun(ctx, project, name)
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
func (mock *MockFreightClient) ListFreightReferencingStage(
	ctx context.Context,
	project, stage string,
) ([]kargoapi.Freight, error) {
	return mock.ListFreightReferencingStageFun(ctx, project, stage)
}
