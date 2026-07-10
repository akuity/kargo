package builtin

import (
	"context"
	"errors"
	"fmt"

	"github.com/akuity/kargo/pkg/promotion/gate/types"
)

// RequestedOriginGateName is the name of the Promotion creation gate that
// enforces that the target Stage requests Freight from the Freight's origin.
const RequestedOriginGateName = "requested-origin"

type requestedOriginGate struct{}

// NewRequestedOriginGate returns a PromotionGate that denies Freight whose
// origin is not requested by the target Stage.
func NewRequestedOriginGate() types.PromotionGate {
	return &requestedOriginGate{}
}

func (g *requestedOriginGate) Name() string {
	return RequestedOriginGateName
}

func (g *requestedOriginGate) Evaluate(
	_ context.Context,
	input types.PromotionInput,
) (*types.Decision, error) {
	if input.Stage == nil {
		return nil, errors.New("stage is nil")
	}
	if input.Freight == nil {
		return nil, errors.New("freight is nil")
	}
	if input.FreightRequest() == nil {
		return types.NewDenyDecision().WithMessage(fmt.Sprintf(
			"Stage %q does not request Freight from %s %q",
			input.Stage.Name,
			input.Freight.Origin.Kind,
			input.Freight.Origin.Name,
		)), nil
	}
	return types.NewAllowDecision(), nil
}
