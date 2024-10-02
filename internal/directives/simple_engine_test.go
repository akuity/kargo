package directives

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestSimpleEngine_Promote(t *testing.T) {
	testHealthCheckStep := HealthCheckStep{
		Kind: "fake",
		Config: Config{
			"fake-key": "fake-value",
		},
	}

	failureResult := PromotionStepResult{Status: PromotionStatusFailed}
	successResult := PromotionStepResult{
		Status:          PromotionStatusSuccess,
		HealthCheckStep: &testHealthCheckStep,
	}

	const successStepName = "success"
	const failureStepName = "failure"
	const contextWaiterStep = "waiter"
	testRegistry := NewStepRunnerRegistry()
	testRegistry.RegisterPromotionStepRunner(
		&mockPromotionStepRunner{
			name:      successStepName,
			runResult: successResult,
		},
		nil,
	)
	testRegistry.RegisterPromotionStepRunner(
		&mockPromotionStepRunner{
			name:      failureStepName,
			runResult: failureResult,
			runErr:    errors.New("something went wrong"),
		},
		nil,
	)
	testRegistry.RegisterPromotionStepRunner(
		&mockPromotionStepRunner{
			name: contextWaiterStep,
			runFunc: func(
				ctx context.Context,
				_ *PromotionStepContext,
			) (PromotionStepResult, error) {
				<-ctx.Done() // Wait for context to be canceled
				return successResult, nil
			},
		},
		nil,
	)

	tests := []struct {
		name       string
		steps      []PromotionStep
		ctx        context.Context
		assertions func(*testing.T, PromotionResult, error)
	}{
		{
			name:  "success: single step",
			steps: []PromotionStep{{Kind: successStepName}},
			ctx:   context.Background(),
			assertions: func(t *testing.T, res PromotionResult, err error) {
				assert.Equal(t, PromotionStatusSuccess, res.Status)
				assert.Equal(t, []HealthCheckStep{testHealthCheckStep}, res.HealthCheckSteps)
				assert.NoError(t, err)
			},
		},
		{
			name: "success: multiple steps",
			steps: []PromotionStep{
				{Kind: successStepName},
				{Kind: successStepName},
			},
			ctx: context.Background(),
			assertions: func(t *testing.T, res PromotionResult, err error) {
				assert.Equal(t, PromotionStatusSuccess, res.Status)
				assert.Equal(
					t,
					[]HealthCheckStep{testHealthCheckStep, testHealthCheckStep},
					res.HealthCheckSteps,
				)
				assert.NoError(t, err)
			},
		},
		{
			name:  "failure: runner not found",
			steps: []PromotionStep{{Kind: "unknown"}},
			ctx:   context.Background(),
			assertions: func(t *testing.T, res PromotionResult, err error) {
				assert.Equal(t, PromotionStatusFailed, res.Status)
				assert.ErrorContains(t, err, "not found")
			},
		},
		{
			name:  "failure: runner returns error",
			steps: []PromotionStep{{Kind: failureStepName}},
			ctx:   context.Background(),
			assertions: func(t *testing.T, res PromotionResult, err error) {
				assert.Equal(t, PromotionStatusFailed, res.Status)
				assert.ErrorContains(t, err, "something went wrong")
			},
		},
		{
			name: "failure: context canceled",
			steps: []PromotionStep{
				{Kind: contextWaiterStep},
				{Kind: contextWaiterStep}, // This runner should not be executed
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
				assert.Equal(t, PromotionStatusFailed, res.Status)
				assert.ErrorIs(t, err, context.Canceled)
			},
		},
	}

	engine := NewSimpleEngine(nil, nil, nil)
	engine.registry = testRegistry

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := engine.Promote(tt.ctx, PromotionContext{}, tt.steps)
			tt.assertions(t, res, err)
		})
	}
}

func TestSimpleEngine_CheckHealth(t *testing.T) {
	testOutput := map[string]any{
		"fake-key": "fake-value",
	}
	testOutputBytes, err := json.Marshal(testOutput)
	require.NoError(t, err)

	const healthyStepName = "healthy"
	const unhealthyStepName = "unhealthy"
	const contextWaiterStep = "waiter"
	testRegistry := NewStepRunnerRegistry()
	testRegistry.RegisterHealthCheckStepRunner(
		&mockHealthCheckStepRunner{
			name: healthyStepName,
			runResult: HealthCheckStepResult{
				Status: kargoapi.HealthStateHealthy,
				Output: testOutput,
			},
		},
		nil,
	)
	testRegistry.RegisterHealthCheckStepRunner(
		&mockHealthCheckStepRunner{
			name: unhealthyStepName,
			runResult: HealthCheckStepResult{
				Status: kargoapi.HealthStateUnhealthy,
				Issues: []string{"something went wrong"},
				Output: testOutput,
			},
		},
		nil,
	)
	testRegistry.RegisterHealthCheckStepRunner(
		&mockHealthCheckStepRunner{
			name: contextWaiterStep,
			runFunc: func(ctx context.Context, _ *HealthCheckStepContext) HealthCheckStepResult {
				<-ctx.Done() // Wait for context to be canceled
				return HealthCheckStepResult{
					Status: kargoapi.HealthStateHealthy,
					Output: testOutput,
				}
			},
		},
		nil,
	)

	tests := []struct {
		name       string
		steps      []HealthCheckStep
		ctx        context.Context
		assertions func(*testing.T, kargoapi.Health)
	}{
		{
			name:  "healthy: single step",
			steps: []HealthCheckStep{{Kind: healthyStepName}},
			ctx:   context.Background(),
			assertions: func(t *testing.T, res kargoapi.Health) {
				require.Equal(t, kargoapi.HealthStateHealthy, res.Status)
				require.Empty(t, res.Issues)
				require.NotNil(t, res.Output)
				require.NotEmpty(t, res.Output.Raw)
				require.Equal(t, 1, strings.Count(string(res.Output.Raw), string(testOutputBytes)))
			},
		},
		{
			name:  "healthy: multiple steps",
			steps: []HealthCheckStep{{Kind: healthyStepName}, {Kind: healthyStepName}},
			ctx:   context.Background(),
			assertions: func(t *testing.T, res kargoapi.Health) {
				require.Equal(t, kargoapi.HealthStateHealthy, res.Status)
				require.Empty(t, res.Issues)
				require.NotNil(t, res.Output)
				require.NotEmpty(t, res.Output.Raw)
				require.Equal(t, 2, strings.Count(string(res.Output.Raw), string(testOutputBytes)))
			},
		},
		{
			name:  "unknown: runner not found",
			steps: []HealthCheckStep{{Kind: healthyStepName}, {Kind: "unknown"}},
			ctx:   context.Background(),
			assertions: func(t *testing.T, res kargoapi.Health) {
				// First step healthy + second step not found == unknown
				require.Equal(t, kargoapi.HealthStateUnknown, res.Status)
				require.Len(t, res.Issues, 1)
				require.Contains(t, res.Issues[0], "no runner registered for step kind")
				require.NotNil(t, res.Output)
				require.NotEmpty(t, res.Output.Raw)
				// We should still get the output from the first step
				require.Equal(t, 1, strings.Count(string(res.Output.Raw), string(testOutputBytes)))
			},
		},
		{
			name:  "unhealthy: a step returns unhealthy",
			steps: []HealthCheckStep{{Kind: unhealthyStepName}, {Kind: healthyStepName}},
			ctx:   context.Background(),
			assertions: func(t *testing.T, res kargoapi.Health) {
				// First step unhealthy + second step unhealthy == unhealthy
				require.Equal(t, kargoapi.HealthStateUnhealthy, res.Status)
				require.Len(t, res.Issues, 1)
				require.Equal(t, "something went wrong", res.Issues[0])
				require.NotNil(t, res.Output)
				require.NotEmpty(t, res.Output.Raw)
				// We should still get the output from both steps
				require.Equal(t, 2, strings.Count(string(res.Output.Raw), string(testOutputBytes)))
			},
		},
		{
			name: "unknown: context canceled",
			steps: []HealthCheckStep{
				{Kind: contextWaiterStep},
				{Kind: contextWaiterStep},
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				go func() {
					time.Sleep(10 * time.Millisecond)
					cancel()
				}()
				return ctx
			}(),
			assertions: func(t *testing.T, res kargoapi.Health) {
				// First step healthy + context canceled == unknown
				require.Equal(t, kargoapi.HealthStateUnknown, res.Status)
				require.Len(t, res.Issues, 1)
				require.Equal(t, context.Canceled.Error(), res.Issues[0])
				require.NotNil(t, res.Output)
				require.NotEmpty(t, res.Output.Raw)
				// We should have output from one step
				require.Equal(t, 1, strings.Count(string(res.Output.Raw), string(testOutputBytes)))
			},
		},
	}

	engine := NewSimpleEngine(nil, nil, nil)
	engine.registry = testRegistry

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.assertions(
				t,
				engine.CheckHealth(tt.ctx, HealthCheckContext{}, tt.steps),
			)
		})
	}
}
