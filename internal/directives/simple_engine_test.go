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
		steps        []PromotionStep
		initRegistry func() *StepRunnerRegistry
		ctx          context.Context
		assertions   func(*testing.T, PromotionResult, error)
	}{
		{
			name:  "success: single step",
			steps: []PromotionStep{{Kind: "mock"}},
			initRegistry: func() *StepRunnerRegistry {
				registry := NewStepRunnerRegistry()
				registry.RegisterPromotionStepRunner(
					&mockPromotionStepRunner{
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
			name: "success: multiple steps",
			steps: []PromotionStep{
				{Kind: "mock1"},
				{Kind: "mock2"},
			},
			initRegistry: func() *StepRunnerRegistry {
				registry := NewStepRunnerRegistry()
				registry.RegisterPromotionStepRunner(
					&mockPromotionStepRunner{
						name:      "mock1",
						runResult: successResult,
					},
					nil,
				)
				registry.RegisterPromotionStepRunner(
					&mockPromotionStepRunner{
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
			name: "failure: runner not found",
			steps: []PromotionStep{
				{Kind: "unknown"},
			},
			initRegistry: func() *StepRunnerRegistry {
				return NewStepRunnerRegistry()
			},
			ctx: context.Background(),
			assertions: func(t *testing.T, res PromotionResult, err error) {
				assert.Equal(t, PromotionStatusFailure, res.Status)
				assert.ErrorContains(t, err, "not found")
			},
		},
		{
			name: "failure: runner returns error",
			steps: []PromotionStep{
				{Kind: "failing", Alias: "alias1", Config: map[string]any{"key": "value"}},
			},
			initRegistry: func() *StepRunnerRegistry {
				registry := NewStepRunnerRegistry()
				registry.RegisterPromotionStepRunner(
					&mockPromotionStepRunner{
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
			steps: []PromotionStep{
				{Kind: "mock"},
				{Kind: "mock"}, // This runner should not be executed
			},
			initRegistry: func() *StepRunnerRegistry {
				registry := NewStepRunnerRegistry()
				registry.RegisterPromotionStepRunner(
					&mockPromotionStepRunner{
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
			engine := NewSimpleEngine(nil, nil, nil)
			engine.registry = tt.initRegistry()
			res, err := engine.Promote(tt.ctx, PromotionContext{}, tt.steps)
			tt.assertions(t, res, err)
		})
	}
}
