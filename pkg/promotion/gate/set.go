package gate

import (
	"context"
	"errors"
	"fmt"

	"github.com/akuity/kargo/pkg/promotion/gate/types"
)

const setName = "set"

// Set is an ordered collection of Promotion creation gates.
type Set struct {
	gates []types.PromotionGate
}

// NewSet returns a PromotionGate that evaluates the provided gates in
// order.
func NewSet(gates ...types.PromotionGate) types.PromotionGate {
	return &Set{
		gates: append([]types.PromotionGate(nil), gates...),
	}
}

func (s *Set) Name() string {
	return setName
}

func (s *Set) Evaluate(
	ctx context.Context,
	input types.PromotionInput,
) (*types.Decision, error) {
	if s == nil {
		return types.NewAllowDecision(), nil
	}

	if input.Stage == nil {
		return types.NewDenyDecision(), errors.New("stage is nil")
	}

	if input.Freight == nil {
		return types.NewDenyDecision(), errors.New("freight is nil")
	}

	for i, promotionGate := range s.gates {
		if promotionGate == nil {
			return nil, fmt.Errorf(
				"promotion creation gate at index %d is nil",
				i,
			)
		}
		decision, err := promotionGate.Evaluate(ctx, input)
		if err != nil {
			return nil, fmt.Errorf(
				"error evaluating Promotion creation gate %q: %w",
				promotionGate.Name(),
				err,
			)
		}
		if !decision.Allow {
			return decision, err
		}

		if decision == nil {
			return types.NewDenyDecision(), errors.New("decision is nil")
		}
	}
	return types.NewAllowDecision(), nil
}
