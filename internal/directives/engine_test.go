package directives

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEngine_Execute(t *testing.T) {
	tests := []struct {
		name         string
		directives   []Directive
		initRegistry func() StepRegistry
		ctx          context.Context
		assertions   func(t *testing.T, result Result, err error)
	}{
		{
			name: "success: single directive",
			directives: []Directive{
				{Step: "mock"},
			},
			initRegistry: func() StepRegistry {
				registry := make(StepRegistry)
				registry.RegisterStep(&mockStep{
					name:      "mock",
					runResult: ResultSuccess,
				})
				return registry
			},
			ctx: context.Background(),
			assertions: func(t *testing.T, result Result, err error) {
				assert.Equal(t, ResultSuccess, result)
				assert.NoError(t, err)
			},
		},
		{
			name: "success: multiple directives",
			directives: []Directive{
				{Step: "mock1"},
				{Step: "mock2"},
			},
			initRegistry: func() StepRegistry {
				registry := make(StepRegistry)
				registry.RegisterStep(&mockStep{
					name:      "mock1",
					runResult: ResultSuccess,
				})
				registry.RegisterStep(&mockStep{
					name:      "mock2",
					runResult: ResultSuccess,
				})
				return registry
			},
			ctx: context.Background(),
			assertions: func(t *testing.T, result Result, err error) {
				assert.Equal(t, ResultSuccess, result)
				assert.NoError(t, err)
			},
		},
		{
			name: "failure: step not found",
			directives: []Directive{
				{Step: "unknown"},
			},
			initRegistry: func() StepRegistry {
				return make(StepRegistry)
			},
			ctx: context.Background(),
			assertions: func(t *testing.T, result Result, err error) {
				assert.Equal(t, ResultFailure, result)
				assert.ErrorContains(t, err, "not found")
			},
		},
		{
			name: "failure: step returns error",
			directives: []Directive{
				{Step: "failing", Alias: "alias1", Config: map[string]any{"key": "value"}},
			},
			initRegistry: func() StepRegistry {
				registry := make(StepRegistry)
				registry.RegisterStep(&mockStep{
					name:      "failing",
					runResult: ResultFailure,
					runErr:    errors.New("something went wrong"),
				})
				return registry
			},
			ctx: context.Background(),
			assertions: func(t *testing.T, result Result, err error) {
				assert.Equal(t, ResultFailure, result)
				assert.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "failure: context canceled",
			directives: []Directive{
				{Step: "mock"},
				{Step: "mock"}, // This step should not be executed
			},
			initRegistry: func() StepRegistry {
				registry := make(StepRegistry)
				registry.RegisterStep(&mockStep{
					name: "mock",
					runFunc: func(ctx context.Context, _ *StepContext) (Result, error) {
						<-ctx.Done() // Wait for context to be canceled
						return ResultSuccess, nil
					},
				})
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
			assertions: func(t *testing.T, result Result, err error) {
				assert.Equal(t, ResultFailure, result)
				assert.ErrorIs(t, err, context.Canceled)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine(tt.initRegistry())
			result, err := engine.Execute(tt.ctx, tt.directives)
			tt.assertions(t, result, err)
		})
	}
}
