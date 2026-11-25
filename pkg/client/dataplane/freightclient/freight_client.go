package freightclient

import (
	"context"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// ListFreightFromOriginOptions is a struct that can be used to specify filtering
// criteria when listing Freight resources that originated from a Warehouse.
//
// +protobuf=false
// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
type ListFreightFromOriginOptions struct {
	// ApprovedFor names a Stage for which all Freight resources that have been
	// approved for that Stage should be included in the list results.
	//
	// IMPORTANT: This is OR'ed with the VerifiedIn field.
	ApprovedFor string
	// VerifiedIn names zero or more Stages for which all Freight resources that
	// have been verified for those Stages should be included in the list results
	//
	// IMPORTANT: This is OR'ed with the ApprovedFor field.
	VerifiedIn []string
	// RequiredSoakTime optionally specifies a minimum duration that a piece of
	// Freight must have continuously remained in a Stage at any time after being
	// verified.
	RequiredSoakTime *metav1.Duration
	// AvailabilityStrategy specifies the semantics for how Freight is determined
	// to be available. If not set, the default is to consider Freight available
	// if it has been verified in any of the provided VerifiedIn stages.
	AvailabilityStrategy kargoapi.FreightAvailabilityStrategy

	// CurrentlyIn optionally specifies a Stage in which the Freight must
	// currently be used in order to be included in the list results.
	CurrentlyIn string
}

// FreightLister is a subset of FreightClient used for ListFreightAvailableToStage
type FreightLister interface {
	ListFreightFromOrigin(
		ctx context.Context,
		project string,
		origin kargoapi.FreightOrigin,
		opts *ListFreightFromOriginOptions,
	) ([]kargoapi.Freight, error)
}

// FreightClient lists Freight through the control plane's API server, so that
// availability can be computed from a component that reaches Freight only
// through the API server rather than the Kubernetes client. *apiserver.Client
// satisfies it.

type FreightClient interface {
	GetFreight(ctx context.Context, project, name string) (*kargoapi.Freight, error)
	GetFreightByAlias(ctx context.Context, project, alias string) (*kargoapi.Freight, error)
	GetFreightByNameOrAlias(ctx context.Context, project, nameOrAlias string) (*kargoapi.Freight, error)

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
		metadata map[string]apiextensionsv1.JSON,
	) error

	ApproveFreight(
		ctx context.Context,
		project, stageName, nameOrAlias string) error

	ListFreightFromOrigin(
		ctx context.Context,
		project string,
		origin kargoapi.FreightOrigin,
		opts *ListFreightFromOriginOptions,
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
