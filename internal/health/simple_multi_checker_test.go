package health

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func TestSimpleMultiChecker_Check(t *testing.T) {
	tests := []struct {
		name       string
		criteria   []Criteria
		assertions func(*testing.T, kargoapi.Health)
	}{
		{
			name: "successful health check",
			criteria: []Criteria{
				{Kind: "success-check"},
			},
			assertions: func(t *testing.T, health kargoapi.Health) {
				assert.Equal(t, kargoapi.HealthStateHealthy, health.Status)
				assert.Empty(t, health.Issues)
				assert.NotNil(t, health.Output)
				assert.JSONEq(t, `[{"test":"success"}]`, string(health.Output.Raw))
			},
		},
		{
			name: "multiple successful health checks",
			criteria: []Criteria{
				{Kind: "success-check"},
				{Kind: "success-check"},
			},
			assertions: func(t *testing.T, health kargoapi.Health) {
				assert.Equal(t, kargoapi.HealthStateHealthy, health.Status)
				assert.Empty(t, health.Issues)
				assert.NotNil(t, health.Output)
				assert.JSONEq(t, `[{"test":"success"},{"test":"success"}]`, string(health.Output.Raw))
			},
		},
		{
			name: "failed health check",
			criteria: []Criteria{
				{Kind: "error-check"},
			},
			assertions: func(t *testing.T, health kargoapi.Health) {
				assert.Equal(t, kargoapi.HealthStateUnhealthy, health.Status)
				assert.Contains(t, health.Issues, "health check failed")
				assert.NotNil(t, health.Output)
			},
		},
		{
			name: "context cancellation",
			criteria: []Criteria{
				{Kind: "context-waiter"},
			},
			assertions: func(t *testing.T, health kargoapi.Health) {
				assert.Equal(t, kargoapi.HealthStateUnknown, health.Status)
				assert.Contains(t, health.Issues, context.Canceled.Error())
				assert.Nil(t, health.Output)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			testRegistry := checkerRegistry{}
			testRegistry.register(
				&mockChecker{
					name: "success-check",
					checkResult: Result{
						Status: kargoapi.HealthStateHealthy,
						Output: map[string]any{"test": "success"},
					},
				},
			)
			testRegistry.register(
				&mockChecker{
					name: "error-check",
					checkResult: Result{
						Status: kargoapi.HealthStateUnhealthy,
						Issues: []string{"health check failed"},
						Output: map[string]any{"test": "error"},
					},
				},
			)
			testRegistry.register(
				&mockChecker{
					name: "context-waiter",
					checkFunc: func(ctx context.Context, _, _ string, _ Criteria) Result {
						cancel()
						<-ctx.Done()
						return Result{
							Status: kargoapi.HealthStateUnknown,
							Issues: []string{ctx.Err().Error()},
						}
					},
				},
			)

			checker := &simpleMultiChecker{
				registry: testRegistry,
			}

			health := checker.Check(ctx, "fake-project", "fake-stage", tt.criteria)
			tt.assertions(t, health)
		})
	}
}

func TestSimpleMultiChecker_executeHealthChecks(t *testing.T) {
	tests := []struct {
		name       string
		criteria   []Criteria
		assertions func(*testing.T, kargoapi.HealthState, []string, []map[string]any)
	}{
		{
			name: "aggregate multiple healthy checks",
			criteria: []Criteria{
				{Kind: "success-check"},
				{Kind: "success-check"},
			},
			assertions: func(t *testing.T, status kargoapi.HealthState, issues []string, output []map[string]any) {
				assert.Equal(t, kargoapi.HealthStateHealthy, status)
				assert.Empty(t, issues)
				assert.Len(t, output, 2)
				for _, o := range output {
					assert.Equal(t, "success", o["test"])
				}
			},
		},
		{
			name: "merge different health states",
			criteria: []Criteria{
				{Kind: "success-check"},
				{Kind: "error-check"},
			},
			assertions: func(t *testing.T, status kargoapi.HealthState, issues []string, output []map[string]any) {
				assert.Equal(t, kargoapi.HealthStateUnhealthy, status)
				assert.Contains(t, issues, "health check failed")
				assert.Len(t, output, 2)
			},
		},
		{
			name: "context cancellation",
			criteria: []Criteria{
				{Kind: "context-waiter"},
				{Kind: "success-check"}, // Should not execute
			},
			assertions: func(t *testing.T, status kargoapi.HealthState, issues []string, output []map[string]any) {
				assert.Equal(t, kargoapi.HealthStateUnknown, status)
				assert.Contains(t, issues, context.Canceled.Error())
				assert.Empty(t, output)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			testRegistry := checkerRegistry{}
			testRegistry.register(
				&mockChecker{
					name: "success-check",
					checkResult: Result{
						Status: kargoapi.HealthStateHealthy,
						Output: map[string]any{"test": "success"},
					},
				},
			)
			testRegistry.register(
				&mockChecker{
					name: "error-check",
					checkResult: Result{
						Status: kargoapi.HealthStateUnhealthy,
						Issues: []string{"health check failed"},
						Output: map[string]any{"test": "error"},
					},
				},
			)
			testRegistry.register(
				&mockChecker{
					name: "context-waiter",
					checkFunc: func(ctx context.Context, _, _ string, _ Criteria) Result {
						cancel()
						<-ctx.Done()
						return Result{
							Status: kargoapi.HealthStateUnknown,
							Issues: []string{ctx.Err().Error()},
						}
					},
				},
			)

			checker := &simpleMultiChecker{
				registry: testRegistry,
			}

			status, issues, output := checker.executeHealthChecks(
				ctx,
				"fake-project",
				"fake-stage",
				tt.criteria,
			)
			tt.assertions(t, status, issues, output)
		})
	}
}

func TestSimpleMultiChecker_executeHealthCheck(t *testing.T) {
	tests := []struct {
		name       string
		criteria   Criteria
		assertions func(*testing.T, Result)
	}{
		{
			name:     "successful execution",
			criteria: Criteria{Kind: "success-check"},
			assertions: func(t *testing.T, result Result) {
				assert.Equal(t, kargoapi.HealthStateHealthy, result.Status)
				assert.Empty(t, result.Issues)
			},
		},
		{
			name:     "unregistered runner",
			criteria: Criteria{Kind: "unknown"},
			assertions: func(t *testing.T, result Result) {
				assert.Equal(t, kargoapi.HealthStateUnknown, result.Status)
				assert.Contains(t, result.Issues[0], "no health checker registered for health check kind")
				assert.Contains(t, result.Issues[0], "unknown")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRegistry := checkerRegistry{}
			testRegistry.register(
				&mockChecker{
					name: "success-check",
					checkResult: Result{
						Status: kargoapi.HealthStateHealthy,
					},
				},
			)

			checker := &simpleMultiChecker{
				registry: testRegistry,
			}

			result := checker.executeHealthCheck(
				context.Background(),
				"fake-project",
				"fake-stage",
				tt.criteria,
			)
			tt.assertions(t, result)
		})
	}
}
