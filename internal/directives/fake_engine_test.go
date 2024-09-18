package directives

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFakeEngine_Execute(t *testing.T) {
	t.Run("without function injection", func(t *testing.T) {
		engine := &FakeEngine{}
		status, err := engine.Execute(context.Background(), PromotionContext{}, nil)
		assert.NoError(t, err)
		assert.Equal(t, StatusSuccess, status)
	})

	t.Run("with function injection", func(t *testing.T) {
		ctx := context.Background()
		promoCtx := PromotionContext{
			Stage: "foo",
		}
		steps := []Step{
			{Directive: "mock"},
		}

		engine := &FakeEngine{
			ExecuteFn: func(givenCtx context.Context, givenpromoCtx PromotionContext, givenSteps []Step) (Status, error) {
				assert.Equal(t, ctx, givenCtx)
				assert.Equal(t, promoCtx, givenpromoCtx)
				assert.Equal(t, steps, givenSteps)
				return StatusFailure, errors.New("something went wrong")
			},
		}
		status, err := engine.Execute(ctx, promoCtx, steps)
		assert.ErrorContains(t, err, "something went wrong")
		assert.Equal(t, StatusFailure, status)
	})
}
