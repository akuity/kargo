package promotion

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
		res, err := engine.Promote(context.Background(), Context{}, nil)
		assert.NoError(t, err)
		assert.Equal(t, kargoapi.PromotionPhaseSucceeded, res.Status)
	})

	t.Run("with function injection", func(t *testing.T) {
		ctx := context.Background()
		promoCtx := Context{
			Stage: "foo",
		}
		steps := []Step{{Kind: "mock"}}

		engine := &FakeEngine{
			PromoteFn: func(
				givenCtx context.Context,
				givenPromoCtx Context,
				givenSteps []Step,
			) (Result, error) {
				assert.Equal(t, ctx, givenCtx)
				assert.Equal(t, promoCtx, givenPromoCtx)
				assert.Equal(t, steps, givenSteps)
				return Result{Status: kargoapi.PromotionPhaseErrored},
					errors.New("something went wrong")
			},
		}
		res, err := engine.Promote(ctx, promoCtx, steps)
		assert.ErrorContains(t, err, "something went wrong")
		assert.Equal(t, kargoapi.PromotionPhaseErrored, res.Status)
	})
}
