package directives

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEngine_Execute(t *testing.T) {
	failureResult := Result{Status: StatusFailure}
	successResult := Result{Status: StatusSuccess}
	tests := []struct {
		name         string
		directives   []Step
		initRegistry func() DirectiveRegistry
		ctx          context.Context
		assertions   func(t *testing.T, status Status, err error)
	}{
		{
			name: "success: single directive",
			directives: []Step{
				{Directive: "mock"},
			},
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
			assertions: func(t *testing.T, status Status, err error) {
				assert.Equal(t, StatusSuccess, status)
				assert.NoError(t, err)
			},
		},
		{
			name: "success: multiple directives",
			directives: []Step{
				{Directive: "mock1"},
				{Directive: "mock2"},
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
			assertions: func(t *testing.T, status Status, err error) {
				assert.Equal(t, StatusSuccess, status)
				assert.NoError(t, err)
			},
		},
		{
			name: "failure: directive not found",
			directives: []Step{
				{Directive: "unknown"},
			},
			initRegistry: func() DirectiveRegistry {
				return make(DirectiveRegistry)
			},
			ctx: context.Background(),
			assertions: func(t *testing.T, status Status, err error) {
				assert.Equal(t, StatusFailure, status)
				assert.ErrorContains(t, err, "not found")
			},
		},
		{
			name: "failure: directive returns error",
			directives: []Step{
				{Directive: "failing", Alias: "alias1", Config: map[string]any{"key": "value"}},
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
			assertions: func(t *testing.T, status Status, err error) {
				assert.Equal(t, StatusFailure, status)
				assert.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "failure: context canceled",
			directives: []Step{
				{Directive: "mock"},
				{Directive: "mock"}, // This directive should not be executed
			},
			initRegistry: func() DirectiveRegistry {
				registry := make(DirectiveRegistry)
				registry.RegisterDirective(
					&mockDirective{
						name: "mock",
						runFunc: func(ctx context.Context, _ *StepContext) (Result, error) {
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
			assertions: func(t *testing.T, status Status, err error) {
				assert.Equal(t, StatusFailure, status)
				assert.ErrorIs(t, err, context.Canceled)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine(tt.initRegistry(), nil, nil, nil)
			status, err := engine.Execute(tt.ctx, tt.directives)
			tt.assertions(t, status, err)
		})
	}
}
