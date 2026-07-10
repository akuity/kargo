package gate

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion/gate/builtin"
	"github.com/akuity/kargo/pkg/promotion/gate/types"
)

// Default composes the standard gate set: static eligibility (namespace,
// requested origin, availability) followed by dynamic soak time.
func Default() types.PromotionGate {
	return NewSet(
		builtin.NewNamespaceGate(),
		builtin.NewRequestedOriginGate(),
		builtin.NewAvailabilityGate(),
		builtin.NewSoakTimeGate(),
	)
}

// DefaultEvaluate runs the default gate for a Stage/Freight.
func DefaultEvaluate(
	ctx context.Context,
	stage *kargoapi.Stage,
	freight *kargoapi.Freight,
) (*types.Decision, error) {
	return Default().Evaluate(
		ctx,
		types.PromotionInput{
			Stage:   stage,
			Freight: freight,
		},
	)
}
