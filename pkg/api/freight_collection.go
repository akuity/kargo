package api

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// NewFreightCollectionForStage builds the FreightCollection a Stage would be
// running once the given Freight is promoted to it: the Freight itself, plus
// the Freight from each of the Stage's other requested origins, carried over
// unchanged from what the Stage is running now.
//
// Carried-over Freight comes from the Stage's last Promotion when there is one
// with a collection, and otherwise from the Stage's freight history. The last
// Promotion is preferred so that Promotions to an ordinary Stage keep building
// on one another exactly as they always have. A Stage that promotes through
// PromotionRequests never records a last Promotion (its Promotions are children
// of a request and take no part in its own flow), so for it the history is the
// source; without that fallback a multi-origin Stage's collection would name
// only the promoted origin and the Stage would appear to drop the rest.
//
// Origins the Stage no longer requests are not carried over, so the result
// never contains Freight the Stage has stopped asking for. The same function
// builds the collection for a Promotion and for a PromotionRequest, so a
// request and its child Promotions agree by construction.
func NewFreightCollectionForStage(
	stage *kargoapi.Stage,
	freight kargoapi.FreightReference,
) *kargoapi.FreightCollection {
	collection := &kargoapi.FreightCollection{}
	if len(stage.Spec.RequestedFreight) > 1 {
		if inherited := inheritableFreight(stage); inherited != nil {
			for _, req := range stage.Spec.RequestedFreight {
				if carried, ok := inherited[req.Origin.String()]; ok {
					collection.UpdateOrPush(carried)
				}
			}
		}
	}
	collection.UpdateOrPush(freight)
	return collection
}

// inheritableFreight returns the Freight, by origin, that a new collection for
// the Stage carries over: the last Promotion's collection when there is one,
// otherwise the Stage's current collection. Nil when there is neither.
func inheritableFreight(stage *kargoapi.Stage) map[string]kargoapi.FreightReference {
	lastRequest := stage.Status.LastPromotionRequest
	lastPromotion := stage.Status.LastPromotion
	if IsTargetAware(stage) && lastRequest != nil &&
		lastRequest.FreightCollection != nil &&
		lastRequest.FreightCollection.Freight != nil {
		return lastRequest.FreightCollection.Freight
	}
	if lastPromotion != nil &&
		lastPromotion.Status != nil &&
		lastPromotion.Status.FreightCollection != nil &&
		lastPromotion.Status.FreightCollection.Freight != nil {
		return lastPromotion.Status.FreightCollection.Freight
	}
	if current := stage.Status.FreightHistory.Current(); current != nil {
		return current.Freight
	}
	return nil
}

// PreparePromotionFreightRefs returns FreightReference and FreightCollection for a promotion
// If Status.Freight and Status.FreightCollection already set, it just returns them
// If the stage is target aware - freight references are taken from PromotionRequest owning the promotion
// If it's a regular stage - freight references are constructed from targetFreight and stage status
// It returns errors if:
// targetFreight is nil
// targetFreight is not yet available for the stage
// for PromotionRequest Status.Freight or Status.FreightCollection are not set yet
// targetFreight is not the freight referenced in PromotionRequest
func PreparePromotionFreightRefs(
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
	// Freight references already resolved
	if promo.Status.Freight != nil && promo.Status.FreightCollection != nil {
		return promo.Status.Freight, promo.Status.FreightCollection, nil
	}
	if IsTargetAware(stage) {
		targetFreightRef, freightCollection, err := GetPromotionRequestFreightRefs(ctx, cli, promo)
		if err != nil {
			return nil, nil, err
		}
		// PromotionRequest supposed to set their freight values by now,
		// but it may not propagated though the caches.
		// Since promotion is already running, the reconciler will restart
		// and hopefully pick up the PromotionRequest changes.
		if targetFreight == nil || freightCollection == nil {
			return nil,
				nil,
				fmt.Errorf("missing freight reference in PromotionRequest for Promotion %q in namespace %q",
					promo.Name, promo.Namespace)
		}
		if targetFreightRef.Name != targetFreight.Name {
			return nil,
				nil,
				fmt.Errorf("freight mismatch between PromotionRequest and Promotion %q in namespace %q",
					promo.Name, promo.Namespace)
		}
		return targetFreightRef, freightCollection, nil
	}
	if !stage.IsFreightAvailable(targetFreight) {
		// nolint:staticcheck
		return nil, nil, fmt.Errorf(
			"Freight %q is not available to Stage %q in namespace %q",
			promo.Spec.Freight,
			stage.Name,
			stage.Namespace,
		)
	}
	targetFreightRef := kargoapi.FreightReference{
		Name:      targetFreight.Name,
		Commits:   targetFreight.Commits,
		Images:    targetFreight.Images,
		Charts:    targetFreight.Charts,
		Artifacts: targetFreight.Artifacts,
		Origin:    targetFreight.Origin,
	}
	freightCollection := NewFreightCollectionForStage(stage, targetFreightRef)
	return &targetFreightRef, freightCollection, nil
}

func GetPromotionRequestFreightRefs(
	ctx context.Context,
	cli client.Client,
	promo kargoapi.Promotion,
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
	return request.Status.Freight, request.Status.FreightCollection, nil
}
