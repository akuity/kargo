package freightclient

import (
	"context"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/api/freightapi"
)

// FreightClient lists Freight through the control plane's API server, so that
// availability can be computed from a component that reaches Freight only
// through the API server rather than the Kubernetes client. *apiserver.Client
// satisfies it.
type FreightClient interface {
	// GetFreight returns the Freight with the given name, or nil if there is no
	// such Freight. Matched by name.
	GetFreight(ctx context.Context, project, name string) (*kargoapi.Freight, error)

	CreateFreight(
		ctx context.Context,
		project string,
		freight *kargoapi.Freight,
		metadata map[string]apiextensionsv1.JSON,
	) (*kargoapi.Freight, bool, error)

	DeleteFreight(ctx context.Context, project, nameOrAlias string) error

	// PatchFreightAlias sets the alias of the named Freight.
	PatchFreightAlias(ctx context.Context, project, nameOrAlias, newAlias string) error

	// PatchFreightMetadata upserts keys in the status metadata of the named
	// Freight, overwriting any value already recorded under the same key.
	PatchFreightMetadata(
		ctx context.Context,
		project, freightName string,
		metadata map[string]any,
	) error

	ListFreightFromWarehouse(
		ctx context.Context,
		project, warehouse string,
		opts *freightapi.ListWarehouseFreightOptions,
	) ([]kargoapi.Freight, error)

	// ListFreightByCurrentStage returns the Freight whose status records it as
	// currently in use by the Stage.
	ListFreightByCurrentStage(
		ctx context.Context,
		project, stage string,
	) ([]kargoapi.Freight, error)

	// ListFreightReferencingStage returns the Freight whose status records the
	// Stage in its verifiedIn or approvedFor map, for cleanup on Stage deletion.
	ListFreightReferencingStage(
		ctx context.Context,
		project, stage string,
	) ([]kargoapi.Freight, error)

	// MarkFreightCurrentlyInStage records that the named Freight, which must
	// have the given origin, is currently in use by the Stage, and that any
	// other Freight from that origin no longer is.
	MarkFreightCurrentlyInStage(
		ctx context.Context,
		project, freight string,
		origin kargoapi.FreightOrigin,
		stage string,
	) error
	// MarkFreightVerifiedInStage records that the named Freight has been
	// verified in the Stage at verifiedAt (or, if zero, when the API server
	// handles the request).
	MarkFreightVerifiedInStage(
		ctx context.Context,
		project, freight, stage string,
		verifiedAt time.Time,
	) error
	// ClearFreightStageStatus removes the Stage from every one of the Freight's
	// status maps (currentlyIn, verifiedIn, approvedFor), for cleanup when the
	// Stage is deleted.
	ClearFreightStageStatus(ctx context.Context, project, freight, stage string) error
}
