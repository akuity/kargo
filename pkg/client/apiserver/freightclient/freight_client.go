package freightclient

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// ListWarehouseFreightOptions is a struct that can be used to specify filtering
// criteria when listing Freight resources that originated from a Warehouse.
//
// +protobuf=false
// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
type ListWarehouseFreightOptions struct {
	// ApprovedFor names a Stage for which all Freight resources that have been
	// approved for that Stage should be included in the list results.
	//
	// IMPORTANT: This is OR'ed with the VerifiedIn field.
	ApprovedFor string
	// VerifiedIn names zero or more Stages for which all Freight resources that
	// have been verified for those Stages should be included in the list results
	// AS long as they were verified before the VerifiedBefore time (if set).
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
	// IMPORTANT: This is also applied to Freight matched using the VerifiedBefore
	// condition.
	AvailabilityStrategy kargoapi.FreightAvailabilityStrategy

	// CurrentlyIn optionally specifies a Stage in which the Freight must
	// currently be used in order to be included in the list results.
	CurrentlyIn string
}

// FreightClient lists Freight through the control plane's API server, so that
// availability can be computed from a component that reaches Freight only
// through the API server rather than the Kubernetes client. *apiserver.Client
// satisfies it.
type FreightClient interface {
	ListFreightFromWarehouse(
		ctx context.Context,
		project, warehouse string,
		opts *ListWarehouseFreightOptions,
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
