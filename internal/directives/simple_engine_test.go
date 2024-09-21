package directives

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEngine_Execute(t *testing.T) {
	failureResult := PromotionStepResult{Status: PromotionStatusFailure}
	successResult := PromotionStepResult{Status: PromotionStatusSuccess}
	tests := []struct {
		name         string
		directives   []PromotionStep
		initRegistry func() DirectiveRegistry
		ctx          context.Context
		assertions   func(*testing.T, PromotionResult, error)
	}{
		{
			name:       "success: single directive",
			directives: []PromotionStep{{Kind: "mock"}},
			initRegistry: func() DirectiveRegistry {
				registry := make(DirectiveRegistry)
				registry.RegisterDirective(
					&mockDirective{
						name:      "mock",
						runResult: successResult,
					},
					nil,
				)
				return registry
			},
			ctx: context.Background(),
			assertions: func(t *testing.T, res PromotionResult, err error) {
				assert.Equal(t, PromotionStatusSuccess, res.Status)
				assert.NoError(t, err)
			},
		},
		{
			name: "success: multiple directives",
			directives: []PromotionStep{
				{Kind: "mock1"},
				{Kind: "mock2"},
			},
			initRegistry: func() DirectiveRegistry {
				registry := make(DirectiveRegistry)
				registry.RegisterDirective(
					&mockDirective{
						name:      "mock1",
						runResult: successResult,
					},
					nil,
				)
				registry.RegisterDirective(
					&mockDirective{
						name:      "mock2",
						runResult: successResult,
					},
					nil,
				)
				return registry
			},
			ctx: context.Background(),
			assertions: func(t *testing.T, res PromotionResult, err error) {
				assert.Equal(t, PromotionStatusSuccess, res.Status)
				assert.NoError(t, err)
			},
		},
		{
			name: "failure: directive not found",
			directives: []PromotionStep{
				{Kind: "unknown"},
			},
			initRegistry: func() DirectiveRegistry {
				return make(DirectiveRegistry)
			},
			ctx: context.Background(),
			assertions: func(t *testing.T, res PromotionResult, err error) {
				assert.Equal(t, PromotionStatusFailure, res.Status)
				assert.ErrorContains(t, err, "not found")
			},
		},
		{
			name: "failure: directive returns error",
			directives: []PromotionStep{
				{Kind: "failing", Alias: "alias1", Config: map[string]any{"key": "value"}},
			},
			initRegistry: func() DirectiveRegistry {
				registry := make(DirectiveRegistry)
				registry.RegisterDirective(
					&mockDirective{
						name:      "failing",
						runResult: failureResult,
						runErr:    errors.New("something went wrong"),
					},
					nil,
				)
				return registry
			},
			ctx: context.Background(),
			assertions: func(t *testing.T, res PromotionResult, err error) {
				assert.Equal(t, PromotionStatusFailure, res.Status)
				assert.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "failure: context canceled",
			directives: []PromotionStep{
				{Kind: "mock"},
				{Kind: "mock"}, // This directive should not be executed
			},
			initRegistry: func() DirectiveRegistry {
				registry := make(DirectiveRegistry)
				registry.RegisterDirective(
					&mockDirective{
						name: "mock",
						runFunc: func(ctx context.Context, _ *PromotionStepContext) (PromotionStepResult, error) {
							<-ctx.Done() // Wait for context to be canceled
							return successResult, nil
						},
					},
					nil,
				)
				return registry
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				go func() {
					time.Sleep(10 * time.Millisecond)
					cancel()
				}()
				return ctx
			}(),
			assertions: func(t *testing.T, res PromotionResult, err error) {
				assert.Equal(t, PromotionStatusFailure, res.Status)
				assert.ErrorIs(t, err, context.Canceled)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewSimpleEngine(tt.initRegistry(), nil, nil, nil)
			res, err := engine.Promote(tt.ctx, PromotionContext{}, tt.directives)
			tt.assertions(t, res, err)
		})
	}
}
