package api

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

// ResolveFreightRefs returns FreightReference and FreightCollection for a promotion
// If Status.Freight and Status.FreightCollection already set, it just returns them
// If the stage is target aware - freight references are taken from PromotionRequest owning the promotion
// If it's a regular stage - freight references are constructed from targetFreight and stage status
// It returns errors if:
// targetFreight is nil
// targetFreight is not yet available for the stage
// for PromotionRequest Status.Freight or Status.FreightCollection are not set yet
// targetFreight is not the freight referenced in PromotionRequest
func ResolveFreightRefs(
	ctx context.Context,
	cli client.Client,
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
	if stage.IsTargetAware() {
		return GetPromotionRequestFreightRefs(ctx, cli, promo, targetFreight)
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

func GetPromotionRequestFreightRefs(
	ctx context.Context,
	cli client.Client,
	promo kargoapi.Promotion,
	targetFreight *kargoapi.Freight,
) (*kargoapi.FreightReference, *kargoapi.FreightCollection, error) {
	owner := PromotionRequestOwner(&promo)
	request, err := GetPromotionRequest(
		ctx,
		cli,
		types.NamespacedName{
			Namespace: promo.Namespace,
			Name:      owner,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	if request == nil {
		return nil, nil, fmt.Errorf("cannot find PromotionRequest %q for Promotion %q", owner, promo.Name)
	}

	// PromotionRequest supposed to set their freight values by now,
	// but it may not propagated though the caches.
	// Since promotion is already running, the reconciler will restart
	// and hopefully pick up the PromotionRequest changes.
	if request.Status.Freight == nil || request.Status.FreightCollection == nil {
		return nil,
			nil,
			fmt.Errorf("missing freight reference in PromotionRequest for Promotion %q in namespace %q",
				promo.Name, promo.Namespace)
	}
	if request.Status.Freight.Name != targetFreight.Name {
		return nil,
			nil,
			fmt.Errorf("freight mismatch between PromotionRequest and Promotion %q in namespace %q",
				promo.Name, promo.Namespace)
	}
	return request.Status.Freight, request.Status.FreightCollection, nil
}
