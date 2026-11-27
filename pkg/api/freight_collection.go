package api

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

// RolloutFreightGetter reads the FreightCollection a PromotionRollout
// recorded when it started, by the Project and the rollout's id.
type RolloutFreightGetter interface {
	GetPromotionRolloutFreightCollection(
		ctx context.Context,
		project string,
		id string,
	) (*kargoapi.FreightCollection, error)
}

// ResolveFreightRefs returns FreightReference and FreightCollection for a promotion
// If Status.Freight and Status.FreightCollection already set, it just returns them
// If the promotion is the child of a PromotionRollout - the collection is the one the rollout
// recorded when it started, read through rollouts, or targetFreight alone if it recorded none
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
	rollouts RolloutFreightGetter,
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

	targetFreightRef := newFreightReference(targetFreight)
	switch {
	case promo.Spec.PromotionRollout != "":
		// A PromotionRollout's child: the collection the rollout recorded
		// when it started. Such a Promotion is recognized by its spec, since
		// a rollout is not a Kubernetes resource and owns nothing, and must
		// be checked for before the next case: its Stage is target aware,
		// but no PromotionRequest stands behind it.
		freightCol, err := rolloutFreightCollection(ctx, rollouts, promo, targetFreightRef)
		if err != nil {
			return nil, nil, err
		}
		return &targetFreightRef, freightCol, nil
	case stage.IsTargetAware():
		// A PromotionRequest's child: the request already resolved both.
		return GetPromotionRequestFreightRefs(ctx, cli, promo, targetFreight)
	default:
		// A regular Stage's Promotion: what the last Promotion left, plus
		// this Freight.
		previous := lastPromotionFreight(stage)
		freightCol := addFreightToCollection(ctx, stage, previous, targetFreightRef)
		return &targetFreightRef, freightCol, nil
	}
}

// rolloutFreightCollection returns the FreightCollection of the
// PromotionRollout that created the Promotion: the one the rollout recorded
// when it started, which holds the Freight it promotes and, for each other
// origin the Stage requests Freight from, the Freight carried forward from
// the Stage's previous rollout. Those are copies, read whole, so a step that
// refers to another origin's Freight finds it even if that Freight is gone.
//
// A rollout that recorded nothing -- one started before collections were
// recorded -- has a collection of the Freight it promotes alone, and so
// does every rollout when there is nothing to read it through. A recorded
// collection whose Freight for the promoted origin is not the Promotion's
// is an error: the rollout and its child disagree on what is promoted.
func rolloutFreightCollection(
	ctx context.Context,
	rollouts RolloutFreightGetter,
	promo kargoapi.Promotion,
	ref kargoapi.FreightReference,
) (*kargoapi.FreightCollection, error) {
	collection := &kargoapi.FreightCollection{}
	if rollouts == nil {
		collection.UpdateOrPush(ref)
		return collection, nil
	}
	recorded, err := rollouts.GetPromotionRolloutFreightCollection(
		ctx,
		promo.Namespace,
		promo.Spec.PromotionRollout,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"error getting the Freight PromotionRollout %q recorded: %w",
			promo.Spec.PromotionRollout, err,
		)
	}
	if recorded == nil || len(recorded.Freight) == 0 {
		collection.UpdateOrPush(ref)
		return collection, nil
	}
	if own, ok := recorded.Freight[ref.Origin.String()]; !ok || own.Name != ref.Name {
		return nil, fmt.Errorf(
			"PromotionRollout %q recorded no Freight %q from %s",
			promo.Spec.PromotionRollout, ref.Name, ref.Origin.String(),
		)
	}
	// Rebuilt rather than taken as read, so it is keyed and identified as a
	// collection built anywhere else from the same Freight is.
	collection.UpdateOrPush(recorded.References()...)
	return collection, nil
}

// addFreightToCollection returns a new FreightCollection that contains all
// FreightReferences from the previous collection (excepting those that are no
// longer requested by the Stage), plus the provided FreightReference, which
// takes the place of any other from the same origin. The previous collection
// is not modified; when it is nil, the new collection holds only the provided
// FreightReference.
func addFreightToCollection(
	ctx context.Context,
	stage *kargoapi.Stage,
	previous *kargoapi.FreightCollection,
	freight kargoapi.FreightReference,
) *kargoapi.FreightCollection {
	logger := logging.LoggerFromContext(ctx)
	freightCol := &kargoapi.FreightCollection{}

	// We don't simply copy the previous FreightCollection because we want to
	// account for the possibility that some freight contained therein are no
	// longer requested by the Stage.
	if len(stage.Spec.RequestedFreight) > 1 {
		if previous != nil && previous.Freight != nil {
			for _, req := range stage.Spec.RequestedFreight {
				if prevFreight, ok := previous.Freight[req.Origin.String()]; ok {
					freightCol.UpdateOrPush(prevFreight)
				}
			}
		} else {
			logger.Debug("no previous collection to inherit Freight from")
		}
	}
	freightCol.UpdateOrPush(freight)
	return freightCol
}

// lastPromotionFreight returns the FreightCollection that the Stage's last
// Promotion left, or nil if the Stage records none.
func lastPromotionFreight(stage *kargoapi.Stage) *kargoapi.FreightCollection {
	lastPromo := stage.Status.LastPromotion
	if lastPromo == nil || lastPromo.Status == nil {
		return nil
	}
	return lastPromo.Status.FreightCollection
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
