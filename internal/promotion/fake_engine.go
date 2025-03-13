package promotion

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// FakeEngine is a mock implementation of the Engine interface that can be used
// to facilitate unit testing.
type FakeEngine struct {
	PromoteFn func(context.Context, Context, []Step) (Result, error)
}

// Promote implements the Engine interface.
func (e *FakeEngine) Promote(
	ctx context.Context,
	promoCtx Context,
	steps []Step,
) (Result, error) {
	if e.PromoteFn == nil {
		return Result{Status: kargoapi.PromotionPhaseSucceeded}, nil
	}
	return e.PromoteFn(ctx, promoCtx, steps)
}
