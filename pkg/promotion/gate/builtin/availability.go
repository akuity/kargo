package builtin

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion/gate/types"
)

// AvailabilityGateName is the name of the Promotion creation gate that enforces
// that Freight is available to the target Stage via approval, a direct source,
// or verification in the required upstream Stage(s).
const AvailabilityGateName = "availability"

type availabilityGate struct{}

// NewAvailabilityGate returns a PromotionGate that allows Freight that is
// approved for the target Stage, sourced directly, or verified in the upstream
// Stage(s) required by the applicable FreightRequest. It does not evaluate
// dynamic policies such as soak time.
//
// It expects the Freight's origin to be requested by the Stage; compose it
// after NewRequestedOriginGate, which reports an unrequested origin with a more
// specific message.
func NewAvailabilityGate() types.PromotionGate {
	return &availabilityGate{}
}

func (g *availabilityGate) Name() string {
	return AvailabilityGateName
}

func (g *availabilityGate) Evaluate(
	_ context.Context,
	input types.PromotionInput,
) (*types.Decision, error) {
	if input.Stage == nil {
		return nil, errors.New("stage is nil")
	}
	if input.Freight == nil {
		return nil, errors.New("freight is nil")
	}
	stage, freight := input.Stage, input.Freight
	request := input.FreightRequest()
	if request == nil {
		// The Stage does not request Freight from this origin, so the Freight
		// cannot be available regardless of approval. The requested-origin gate
		// reports this case with a more specific message.
		return types.NewDenyDecision().WithMessage(fmt.Sprintf(
			"Freight %q is not available to Stage %q",
			freight.Name,
			stage.Name,
		)), nil
	}
	if freight.IsApprovedFor(stage.Name) {
		return types.NewAllowDecision(), nil
	}
	if request.Sources.Direct {
		return types.NewAllowDecision(), nil
	}
	if isVerifiedUpstream(freight, request.Sources) {
		return types.NewAllowDecision(), nil
	}
	message := unavailableMessage(freight.Name, stage.Name, request.Sources)
	return types.NewDenyDecision().WithMessage(message), errors.New(message)
}

// isVerifiedUpstream reports whether the Freight satisfies the upstream
// verification requirement of the sources, honoring the availability strategy.
func isVerifiedUpstream(
	freight *kargoapi.Freight,
	sources kargoapi.FreightSources,
) bool {
	if sources.AvailabilityStrategy == kargoapi.FreightAvailabilityStrategyAll {
		// Freight must be verified in every upstream Stage.
		for _, upstream := range sources.Stages {
			if !freight.IsVerifiedIn(upstream) {
				return false
			}
		}
		return true
	}
	// Freight must be verified in at least one upstream Stage.
	return slices.ContainsFunc(sources.Stages, freight.IsVerifiedIn)
}

func unavailableMessage(
	freightName string,
	stageName string,
	sources kargoapi.FreightSources,
) string {
	stages := strings.Join(sources.Stages, ", ")
	if sources.AvailabilityStrategy == kargoapi.FreightAvailabilityStrategyAll {
		return fmt.Sprintf(
			"Freight %q must be verified in all upstream Stages (%s) "+
				"to be available to Stage %q",
			freightName,
			stages,
			stageName,
		)
	}
	return fmt.Sprintf(
		"Freight %q must be verified in at least one upstream Stage (%s) "+
			"to be available to Stage %q",
		freightName,
		stages,
		stageName,
	)
}
