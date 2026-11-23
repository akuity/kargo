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

	ListFreightFromOrigin(
		ctx context.Context,
		project string,
		origin kargoapi.FreightOrigin,
		opts *freightapi.ListFreightFromOriginOptions,
	) ([]kargoapi.Freight, error)

	// MarkFreightsCurrentlyInStage records that the set of Freights are currently in use by the Stage,
	// and that any other Freights no longer are.
	MarkFreightsCurrentlyInStage(
		ctx context.Context,
		project, stage string,
		freights []string,
	) error

	// MarkFreightsVerifiedInStage records that the set of Freights are verified in Stage.
	MarkFreightsVerifiedInStage(
		ctx context.Context,
		project, stage string,
		freights []string,
		verifiedAt time.Time,
	) error

	// ClearStageFreights removes references to stage (currentlyIn, approvedFor, verifiedIn)
	// from all freights referencing it.
	ClearStageFreights(
		ctx context.Context,
		project, stage string,
	) error
}
