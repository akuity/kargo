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
	var base *kargoapi.FreightCollection
	if lastPromo := stage.Status.LastPromotion; lastPromo != nil && lastPromo.Status != nil {
		base = lastPromo.Status.FreightCollection
	}
	return buildFreightCollection(ctx, targetFreight, stage, base)
}

// buildFreightCollection constructs a FreightCollection that contains all
// FreightReferences from the provided base collection (excepting those that
// are no longer requested by the Stage), plus a FreightReference for the
// provided targetFreight. With a nil base, the collection holds only
// targetFreight.
func buildFreightCollection(
	ctx context.Context,
	targetFreight kargoapi.FreightReference,
	stage *kargoapi.Stage,
	base *kargoapi.FreightCollection,
) *kargoapi.FreightCollection {
	logger := logging.LoggerFromContext(ctx)
	freightCol := &kargoapi.FreightCollection{}

	// We don't simply copy the base FreightCollection because we want to
	// account for the possibility that some freight contained therein are no
	// longer requested by the Stage.
	if len(stage.Spec.RequestedFreight) > 1 {
		if base != nil && base.Freight != nil {
			for _, req := range stage.Spec.RequestedFreight {
				if freight, ok := base.Freight[req.Origin.String()]; ok {
					freightCol.UpdateOrPush(freight)
				}
			}
		} else {
			logger.Debug("no collection to inherit Freight from")
		}
	}
	freightCol.UpdateOrPush(targetFreight)
	return freightCol
}

// ResolveFreightRefs returns FreightReference and FreightCollection for a promotion
// If Status.Freight and Status.FreightCollection already set, it just returns them
// If the promotion is the child of a PromotionRollout - freight references are constructed from
// targetFreight and the stage's Freight history
// If the stage is otherwise target aware - freight references are taken from PromotionRequest
// owning the promotion
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
	// A PromotionRollout's child is recognized by its spec, not by an owner:
	// a rollout is not a Kubernetes resource and so owns nothing. This must
	// be checked before the Stage is, since such a Promotion's Stage is
	// target aware but no PromotionRequest stands behind the Promotion.
	if promo.Spec.PromotionRollout != "" {
		return getPromotionRolloutFreightRefs(ctx, targetFreight, stage)
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
	targetFreightRef := newFreightReference(targetFreight)
	freightCollection := buildPromotionFreightCollection(ctx, targetFreightRef, stage)
	return &targetFreightRef, freightCollection, nil
}

// getPromotionRolloutFreightRefs returns the FreightReference and
// FreightCollection for a Promotion that a PromotionRollout created as one
// part of its fan-out to a Stage's Targets. The collection contains all
// FreightReferences the Stage is currently on (excepting those that are no
// longer requested), plus a FreightReference for the provided targetFreight.
//
// A rollout's child cannot inherit from the Stage's last Promotion, as a
// regular Stage's Promotion does: a target-aware Stage never records one. It
// inherits from the Stage's Freight history instead. Every child of one
// rollout therefore resolves the same collection for as long as that history
// does not move while the rollout runs, and it moves only when a round of
// promotions to the Stage's Targets finishes.
func getPromotionRolloutFreightRefs(
	ctx context.Context,
	targetFreight *kargoapi.Freight,
	stage *kargoapi.Stage,
) (*kargoapi.FreightReference, *kargoapi.FreightCollection, error) {
	targetFreightRef := newFreightReference(targetFreight)
	freightCollection := buildFreightCollection(
		ctx,
		targetFreightRef,
		stage,
		stage.Status.FreightHistory.Current(),
	)
	return &targetFreightRef, freightCollection, nil
}

// newFreightReference returns a FreightReference to the provided Freight.
func newFreightReference(freight *kargoapi.Freight) kargoapi.FreightReference {
	return kargoapi.FreightReference{
		Name:      freight.Name,
		Commits:   freight.Commits,
		Images:    freight.Images,
		Charts:    freight.Charts,
		Artifacts: freight.Artifacts,
		Origin:    freight.Origin,
	}
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
