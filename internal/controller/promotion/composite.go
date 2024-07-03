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
	newFreight []kargoapi.FreightReference,
) (*kargoapi.PromotionStatus, []kargoapi.FreightReference, error) {
	if stage.Spec.PromotionMechanisms == nil {
		return &kargoapi.PromotionStatus{Phase: kargoapi.PromotionPhaseSucceeded},
			newFreight, nil
	}

	var newStatus *kargoapi.PromotionStatus

	logger := logging.LoggerFromContext(ctx).WithValues("name", c.name)
	logger.Debug("executing composite promotion mechanism")

	for _, childMechanism := range c.childMechanisms {
		var err error
		var otherStatus *kargoapi.PromotionStatus
		otherStatus, newFreight, err = childMechanism.Promote(ctx, stage, promo, newFreight)
		if err != nil {
			return nil, newFreight, fmt.Errorf(
				"error executing %s: %w",
				childMechanism.GetName(),
				err,
			)
		}
		newStatus = aggregateGitPromoStatus(newStatus, *otherStatus)
		if newStatus.Phase != kargoapi.PromotionPhaseSucceeded {
			// We only continue to the next promotion mechanism if the current
			// mechanism succeeded. This is because a PR must be merged before
			// performing the ArgoCD sync.
			break
		}
	}

	logger.Debug(
		"done executing composite promotion mechanism",
		"aggregatedStatus", newStatus.Phase,
	)

	return newStatus, newFreight, nil
}

// aggregateGitPromoStatus returns the aggregated status of two promotion statuses when
// multiple promote mechanisms are used. Returns the most severe phase. In order of precedence:
//
//	Error, Failed, Running, Succeeded
func aggregateGitPromoStatus(curr *kargoapi.PromotionStatus, other kargoapi.PromotionStatus) *kargoapi.PromotionStatus {
	if curr == nil {
		return other.DeepCopy()
	}
	newStatus := curr.DeepCopy()
	if curr.Phase == kargoapi.PromotionPhaseErrored {
		// do nothing. we are already at most severe phase
	} else if other.Phase == kargoapi.PromotionPhaseErrored {
		newStatus.Phase = kargoapi.PromotionPhaseErrored
		newStatus.Message = other.Message
	} else if curr.Phase == kargoapi.PromotionPhaseFailed || other.Phase == kargoapi.PromotionPhaseFailed {
		newStatus.Phase = kargoapi.PromotionPhaseFailed
		newStatus.Message = firstNonEmpty(curr.Message, other.Message)
	} else if curr.Phase == kargoapi.PromotionPhaseRunning || other.Phase == kargoapi.PromotionPhaseRunning {
		newStatus.Phase = kargoapi.PromotionPhaseRunning
		newStatus.Message = firstNonEmpty(curr.Message, other.Message)
	} else {
		newStatus.Phase = kargoapi.PromotionPhaseSucceeded
		newStatus.Message = firstNonEmpty(curr.Message, other.Message)
	}
	// Merge the two metadata maps
	if len(other.Metadata) > 0 {
		if newStatus.Metadata == nil {
			newStatus.Metadata = make(map[string]string)
		}
		for k, v := range other.Metadata {
			newStatus.Metadata[k] = v
		}
	}
	return newStatus
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
