package api

import (
	"context"
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

// buildPromotionFreightCollection constructs a FreightCollection that contains all
// FreightReferences from the previous Promotion (excepting those that are no
// longer requested), plus a FreightReference for the provided targetFreight.
func buildPromotionFreightCollection(
	ctx context.Context,
	targetFreight kargoapi.FreightReference,
	stage *kargoapi.Stage,
) *kargoapi.FreightCollection {
	logger := logging.LoggerFromContext(ctx)
	freightCol := &kargoapi.FreightCollection{}

	// We don't simply copy the current FreightCollection because we want to
	// account for the possibility that some freight contained therein are no
	// longer requested by the Stage.
	if len(stage.Spec.RequestedFreight) > 1 {
		lastPromo := stage.Status.LastPromotion
		if lastPromo != nil && lastPromo.Status != nil && lastPromo.Status.FreightCollection != nil &&
			lastPromo.Status.FreightCollection.Freight != nil {
			for _, req := range stage.Spec.RequestedFreight {
				if freight, ok := lastPromo.Status.FreightCollection.Freight[req.Origin.String()]; ok {
					freightCol.UpdateOrPush(freight)
				}
			}
		} else {
			logger.Debug("last promotion has no collection to inherit Freight from")
		}
	}
	freightCol.UpdateOrPush(targetFreight)
	return freightCol
}

// ResolveFreightRefs returns the FreightReference and FreightCollection for
// a Promotion. If Status.Freight and Status.FreightCollection are already
// set, it just returns them; otherwise they are constructed from
// targetFreight and the Stage's status. A Promotion to one of a Stage's
// Targets resolves them exactly as a Promotion to the Stage itself does: its
// Freight is the Freight of the PromotionRollout that created it.
// It returns errors if:
// targetFreight is nil
// targetFreight is not yet available for the stage
func ResolveFreightRefs(
	ctx context.Context,
	promo kargoapi.Promotion,
	stage *kargoapi.Stage,
	targetFreight *kargoapi.Freight,
) (*kargoapi.FreightReference, *kargoapi.FreightCollection, error) {
	if targetFreight == nil {
		// nolint:staticcheck
		return nil, nil, fmt.Errorf(
			"Freight %q not found in namespace %q",
			promo.Spec.Freight, promo.Namespace,
		)
	}
	// Always check if freight is available for stage
	if !stage.IsFreightAvailable(targetFreight) {
		// nolint:staticcheck
		return nil, nil, fmt.Errorf(
			"Freight %q is not available to Stage %q in namespace %q",
			promo.Spec.Freight,
			stage.Name,
			stage.Namespace,
		)
	}
	// Freight references already resolved
	if promo.Status.Freight != nil && promo.Status.FreightCollection != nil {
		return promo.Status.Freight, promo.Status.FreightCollection, nil
	}
	return GetStageFreightRefs(ctx, targetFreight, stage)
}

func GetStageFreightRefs(
	ctx context.Context,
	targetFreight *kargoapi.Freight,
	stage *kargoapi.Stage,
) (*kargoapi.FreightReference, *kargoapi.FreightCollection, error) {
	targetFreightRef := kargoapi.FreightReference{
		Name:      targetFreight.Name,
		Commits:   targetFreight.Commits,
		Images:    targetFreight.Images,
		Charts:    targetFreight.Charts,
		Artifacts: targetFreight.Artifacts,
		Origin:    targetFreight.Origin,
	}
	freightCollection := buildPromotionFreightCollection(ctx, targetFreightRef, stage)
	return &targetFreightRef, freightCollection, nil
}
