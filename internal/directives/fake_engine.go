package directives

import (
	"context"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// FakeEngine is a mock implementation of the Engine interface that can be used
// to facilitate unit testing.
type FakeEngine struct {
	ExecuteFn     func(context.Context, PromotionContext, []PromotionStep) (PromotionResult, error)
	CheckHealthFn func(context.Context, HealthCheckContext, []HealthCheckStep) kargoapi.Health
}

// Promote implements the Engine interface.
func (e *FakeEngine) Promote(
	ctx context.Context,
	promoCtx PromotionContext,
	steps []PromotionStep,
) (PromotionResult, error) {
	if e.ExecuteFn == nil {
		return PromotionResult{Status: PromotionStatusSucceeded}, nil
	}
	return e.ExecuteFn(ctx, promoCtx, steps)
}

// CheckHealth implements the Engine interface.
func (e *FakeEngine) CheckHealth(
	ctx context.Context,
	healthCtx HealthCheckContext,
	steps []HealthCheckStep,
) kargoapi.Health {
	if e.CheckHealthFn == nil {
		return kargoapi.Health{Status: kargoapi.HealthStateHealthy}
	}
	return e.CheckHealthFn(ctx, healthCtx, steps)
}
