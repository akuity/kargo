package freightclient

import (
	"context"
	"time"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api/freightapi"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// FreightClient lists Freight through the control plane's API server, so that
// availability can be computed from a component that reaches Freight only
// through the API server rather than the Kubernetes client. *apiserver.Client
// satisfies it.
type FreightClient interface {
	CreateFreight(
	ctx context.Context,
	project string,
	freight *kargoapi.Freight,
	metadata map[string]apiextensionsv1.JSON,
	) (*kargoapi.Freight, bool, error)
	DeleteFreight(ctx context.Context, project, nameOrAlias string) error
	ListFreightFromWarehouse(
		ctx context.Context,
		project, warehouse string,
		opts *freightapi.ListWarehouseFreightOptions,
	) ([]kargoapi.Freight, error)
	MarkFreightInStage(
		ctx context.Context,
		project, freight, stage string,
		verified bool,
		verifiedAt time.Time,
	) error
	UnmarkFreightInStage(ctx context.Context, project, freight, stage string) error
	// ClearFreightStageStatus removes the Stage from every one of the Freight's
	// status maps (currentlyIn, verifiedIn, approvedFor), for cleanup when the
	// Stage is deleted.
	ClearFreightStageStatus(ctx context.Context, project, freight, stage string) error
	// ListFreightByCurrentStage returns the Freight whose status records it as
	// currently in use by the Stage.
	ListFreightByCurrentStage(
		ctx context.Context,
		project, stage string,
	) ([]kargoapi.Freight, error)
	// GetFreight returns the named Freight, or nil if there is no such Freight.
	GetFreight(ctx context.Context, project, nameOrAlias string) (*kargoapi.Freight, error)
	// ListFreightReferencingStage returns the Freight whose status records the
	// Stage in its verifiedIn or approvedFor map, for cleanup on Stage deletion.
	ListFreightReferencingStage(
		ctx context.Context,
		project, stage string,
	) ([]kargoapi.Freight, error)
}
