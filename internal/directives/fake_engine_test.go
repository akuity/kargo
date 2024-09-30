package directives

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestFakeEngine_Promote(t *testing.T) {
	t.Run("without function injection", func(t *testing.T) {
		engine := &FakeEngine{}
		res, err := engine.Promote(context.Background(), PromotionContext{}, nil)
		assert.NoError(t, err)
		assert.Equal(t, PromotionStatusSuccess, res.Status)
	})

	t.Run("with function injection", func(t *testing.T) {
		ctx := context.Background()
		promoCtx := PromotionContext{
			Stage: "foo",
		}
		steps := []PromotionStep{{Kind: "mock"}}

		engine := &FakeEngine{
			ExecuteFn: func(
				givenCtx context.Context,
				givenPromoCtx PromotionContext,
				givenSteps []PromotionStep,
			) (PromotionResult, error) {
				assert.Equal(t, ctx, givenCtx)
				assert.Equal(t, promoCtx, givenPromoCtx)
				assert.Equal(t, steps, givenSteps)
				return PromotionResult{Status: PromotionStatusFailure},
					errors.New("something went wrong")
			},
		}
		res, err := engine.Promote(ctx, promoCtx, steps)
		assert.ErrorContains(t, err, "something went wrong")
		assert.Equal(t, PromotionStatusFailure, res.Status)
	})
}

func TestFakeEngine_CheckHealth(t *testing.T) {
	t.Run("without function injection", func(t *testing.T) {
		engine := &FakeEngine{}
		res := engine.CheckHealth(context.Background(), HealthCheckContext{}, nil)
		assert.Equal(t, kargoapi.HealthStateHealthy, res.Status)
	})

	t.Run("with function injection", func(t *testing.T) {
		ctx := context.Background()
		healthCtx := HealthCheckContext{
			Stage: "foo",
		}
		steps := []HealthCheckStep{{Kind: "mock"}}

		engine := &FakeEngine{
			CheckHealthFn: func(
				givenCtx context.Context,
				givenHealthCtx HealthCheckContext,
				givenSteps []HealthCheckStep,
			) kargoapi.Health {
				assert.Equal(t, ctx, givenCtx)
				assert.Equal(t, healthCtx, givenHealthCtx)
				assert.Equal(t, steps, givenSteps)
				return kargoapi.Health{Status: kargoapi.HealthStateUnhealthy}
			},
		}
		res := engine.CheckHealth(ctx, healthCtx, steps)
		assert.Equal(t, kargoapi.HealthStateUnhealthy, res.Status)
	})
}
