package promotion

import (
	"context"
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/logging"
)

// compositeMechanism is an implementation of the Mechanism interface that is
// composed only of other Mechanisms. Executing Promote() or CheckHealth() on a
// compositeMechanism will execute that same function on each of its child
// Mechanisms in turn.
type compositeMechanism struct {
	name            string
	childMechanisms []Mechanism
}

// newCompositeMechanism returns an implementation of the Mechanism interface
// that is composed only of other Mechanisms. Executing Promote() or
// CheckHealth() on a compositeMechanism will execute that same function on each
// of its child Mechanisms in turn.
// Promotions are executed *in sequence* and will not proceed to the next mechanism
// until the mechanism returns Successful
func newCompositeMechanism(
	name string,
	childPromotionMechanisms ...Mechanism,
) Mechanism {
	return &compositeMechanism{
		name:            name,
		childMechanisms: childPromotionMechanisms,
	}
}

// GetName implements the Mechanism interface.
func (c *compositeMechanism) GetName() string {
	return c.name
}

// Promote implements the Mechanism interface.
func (c *compositeMechanism) Promote(
	ctx context.Context,
	stage *kargoapi.Stage,
	promo *kargoapi.Promotion,
) error {
	if stage.Spec.PromotionMechanisms == nil {
		promo.Status.Phase = kargoapi.PromotionPhaseSucceeded
		return nil
	}

	logger := logging.LoggerFromContext(ctx).WithValues("name", c.name)
	logger.Debug("executing composite promotion mechanism")

	// Start with success and degrade as child mechanisms report more severe
	// phases.
	promo.Status.Phase = kargoapi.PromotionPhaseSucceeded
	for _, childMechanism := range c.childMechanisms {
		origStatus := promo.Status.DeepCopy()
		if err := childMechanism.Promote(ctx, stage, promo); err != nil {
			return fmt.Errorf("error executing %s: %w", childMechanism.GetName(), err)
		}
		promo.Status = *mergePromoStatus(&promo.Status, origStatus)
		if promo.Status.Phase != kargoapi.PromotionPhaseSucceeded {
			// We only continue to the next promotion mechanism if the current
			// mechanism succeeded. This is because a PR must be merged before
			// performing the ArgoCD sync.
			break
		}
	}

	logger.Debug(
		"done executing composite promotion mechanism",
		"aggregatedStatus", promo.Status.Phase,
	)

	return nil
}

// mergePromoStatus merges the PromotionStatus represented by newerStatus into a
// deep copy of the PromotionStatus represented by olderStatus and returns the
// result. The returned status will differ from the olderStatus in the following
// ways:
//
//  1. The Phase and corresponding Message are updated to reflect the more
//     severe of the two. The order of severity is:
//     Errored > Failed > Running > Succeeded.
//  2. The FreightCollection is unconditionally updated to that of src.
//  3. Metadata from src is merged into Metadata from olderStatus, with Metadata
//     from src taking precedence in case of key conflicts.
//
// Both arguments must be non-nil.
func mergePromoStatus(
	newerStatus *kargoapi.PromotionStatus,
	olderStatus *kargoapi.PromotionStatus,
) *kargoapi.PromotionStatus {
	mergedStatus := olderStatus.DeepCopy()
	switch {
	case mergedStatus.Phase == kargoapi.PromotionPhaseErrored:
		// Do nothing. We are already at most severe phase.
	case newerStatus.Phase == kargoapi.PromotionPhaseErrored:
		mergedStatus.Phase = kargoapi.PromotionPhaseErrored
		mergedStatus.Message = newerStatus.Message
	case mergedStatus.Phase == kargoapi.PromotionPhaseFailed || newerStatus.Phase == kargoapi.PromotionPhaseFailed:
		mergedStatus.Phase = kargoapi.PromotionPhaseFailed
		mergedStatus.Message = firstNonEmpty(mergedStatus.Message, newerStatus.Message)
	case mergedStatus.Phase == kargoapi.PromotionPhaseRunning || newerStatus.Phase == kargoapi.PromotionPhaseRunning:
		mergedStatus.Phase = kargoapi.PromotionPhaseRunning
		mergedStatus.Message = firstNonEmpty(mergedStatus.Message, newerStatus.Message)
	case mergedStatus.Phase == kargoapi.PromotionPhaseSucceeded && newerStatus.Phase == kargoapi.PromotionPhaseSucceeded:
		mergedStatus.Message = firstNonEmpty(mergedStatus.Message, newerStatus.Message)
	}
	mergedStatus.FreightCollection = newerStatus.FreightCollection
	// Merge the two metadata maps
	if len(newerStatus.Metadata) > 0 {
		if mergedStatus.Metadata == nil {
			mergedStatus.Metadata = make(map[string]string, len(newerStatus.Metadata))
		}
		for k, v := range newerStatus.Metadata {
			mergedStatus.Metadata[k] = v
		}
	}
	return mergedStatus
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
