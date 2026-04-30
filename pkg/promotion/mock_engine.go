package promotion

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// MockEngine is a mock implementation of the Engine interface that can be used
// to facilitate unit testing.
type MockEngine struct {
	PromoteFn func(context.Context, Context, []Step) (Result, error)
}

// Promote implements the Engine interface.
func (m *MockEngine) Promote(
	ctx context.Context,
	promoCtx Context,
	steps []Step,
) (Result, error) {
	if m.PromoteFn == nil {
		return Result{Status: kargoapi.PromotionPhaseSucceeded}, nil
	}
	return m.PromoteFn(ctx, promoCtx, steps)
}
