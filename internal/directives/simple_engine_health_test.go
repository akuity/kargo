package directives

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

func TestSimpleEngine_CheckHealth(t *testing.T) {
	tests := []struct {
		name       string
		healthCtx  HealthCheckContext
		steps      []HealthCheckStep
		assertions func(*testing.T, kargoapi.Health)
	}{
		{
			name: "successful health check",
			steps: []HealthCheckStep{
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
			steps: []HealthCheckStep{
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
			steps: []HealthCheckStep{
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
			steps: []HealthCheckStep{
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

			testRegistry := NewStepRunnerRegistry()
			testRegistry.RegisterHealthCheckStepRunner(
				&mockHealthCheckStepRunner{
					name: "success-check",
					runResult: HealthCheckStepResult{
						Status: kargoapi.HealthStateHealthy,
						Output: State{"test": "success"},
					},
				},
				nil,
			)
			testRegistry.RegisterHealthCheckStepRunner(
				&mockHealthCheckStepRunner{
					name: "error-check",
					runResult: HealthCheckStepResult{
						Status: kargoapi.HealthStateUnhealthy,
						Issues: []string{"health check failed"},
						Output: State{"test": "error"},
					},
				},
				nil,
			)
			testRegistry.RegisterHealthCheckStepRunner(
				&mockHealthCheckStepRunner{
					name: "context-waiter",
					runFunc: func(ctx context.Context, _ *HealthCheckStepContext) HealthCheckStepResult {
						cancel()
						<-ctx.Done()
						return HealthCheckStepResult{
							Status: kargoapi.HealthStateUnknown,
							Issues: []string{ctx.Err().Error()},
						}
					},
				},
				nil,
			)

			engine := &SimpleEngine{
				registry: testRegistry,
			}

			health := engine.CheckHealth(ctx, tt.healthCtx, tt.steps)
			tt.assertions(t, health)
		})
	}
}

func TestSimpleEngine_executeHealthChecks(t *testing.T) {
	tests := []struct {
		name       string
		healthCtx  HealthCheckContext
		steps      []HealthCheckStep
		assertions func(*testing.T, kargoapi.HealthState, []string, []State)
	}{
		{
			name: "aggregate multiple healthy checks",
			steps: []HealthCheckStep{
				{Kind: "success-check"},
				{Kind: "success-check"},
			},
			assertions: func(t *testing.T, status kargoapi.HealthState, issues []string, output []State) {
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
			steps: []HealthCheckStep{
				{Kind: "success-check"},
				{Kind: "error-check"},
			},
			assertions: func(t *testing.T, status kargoapi.HealthState, issues []string, output []State) {
				assert.Equal(t, kargoapi.HealthStateUnhealthy, status)
				assert.Contains(t, issues, "health check failed")
				assert.Len(t, output, 2)
			},
		},
		{
			name: "context cancellation",
			steps: []HealthCheckStep{
				{Kind: "context-waiter"},
				{Kind: "success-check"}, // Should not execute
			},
			assertions: func(t *testing.T, status kargoapi.HealthState, issues []string, output []State) {
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

			testRegistry := NewStepRunnerRegistry()
			testRegistry.RegisterHealthCheckStepRunner(
				&mockHealthCheckStepRunner{
					name: "success-check",
					runResult: HealthCheckStepResult{
						Status: kargoapi.HealthStateHealthy,
						Output: State{"test": "success"},
					},
				},
				nil,
			)
			testRegistry.RegisterHealthCheckStepRunner(
				&mockHealthCheckStepRunner{
					name: "error-check",
					runResult: HealthCheckStepResult{
						Status: kargoapi.HealthStateUnhealthy,
						Issues: []string{"health check failed"},
						Output: State{"test": "error"},
					},
				},
				nil,
			)
			testRegistry.RegisterHealthCheckStepRunner(
				&mockHealthCheckStepRunner{
					name: "context-waiter",
					runFunc: func(ctx context.Context, _ *HealthCheckStepContext) HealthCheckStepResult {
						cancel()
						<-ctx.Done()
						return HealthCheckStepResult{
							Status: kargoapi.HealthStateUnknown,
							Issues: []string{ctx.Err().Error()},
						}
					},
				},
				nil,
			)

			engine := &SimpleEngine{
				registry: testRegistry,
			}

			status, issues, output := engine.executeHealthChecks(ctx, tt.healthCtx, tt.steps)
			tt.assertions(t, status, issues, output)
		})
	}
}

func TestSimpleEngine_executeHealthCheck(t *testing.T) {
	tests := []struct {
		name       string
		healthCtx  HealthCheckContext
		step       HealthCheckStep
		assertions func(*testing.T, HealthCheckStepResult)
	}{
		{
			name: "successful execution",
			step: HealthCheckStep{Kind: "success-check"},
			assertions: func(t *testing.T, result HealthCheckStepResult) {
				assert.Equal(t, kargoapi.HealthStateHealthy, result.Status)
				assert.Empty(t, result.Issues)
			},
		},
		{
			name: "unregistered runner",
			step: HealthCheckStep{Kind: "unknown"},
			assertions: func(t *testing.T, result HealthCheckStepResult) {
				assert.Equal(t, kargoapi.HealthStateUnknown, result.Status)
				assert.Contains(t, result.Issues[0], "no runner registered for step kind")
				assert.Contains(t, result.Issues[0], "unknown")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRegistry := NewStepRunnerRegistry()
			testRegistry.RegisterHealthCheckStepRunner(
				&mockHealthCheckStepRunner{
					name: "success-check",
					runResult: HealthCheckStepResult{
						Status: kargoapi.HealthStateHealthy,
					},
				},
				nil,
			)

			engine := &SimpleEngine{
				registry: testRegistry,
			}

			result := engine.executeHealthCheck(context.Background(), tt.healthCtx, tt.step)
			tt.assertions(t, result)
		})
	}
}

func TestSimpleEngine_prepareHealthCheckStepContext(t *testing.T) {
	tests := []struct {
		name        string
		healthCtx   HealthCheckContext
		step        HealthCheckStep
		permissions StepRunnerPermissions
		assertions  func(*testing.T, *HealthCheckStepContext)
	}{
		{
			name: "context with all permissions",
			healthCtx: HealthCheckContext{
				Project: "test-project",
				Stage:   "test-stage",
			},
			step: HealthCheckStep{
				Config: map[string]any{
					"key": "value",
				},
			},
			permissions: StepRunnerPermissions{
				AllowCredentialsDB: true,
				AllowKargoClient:   true,
				AllowArgoCDClient:  true,
			},
			assertions: func(t *testing.T, ctx *HealthCheckStepContext) {
				assert.Equal(t, "test-project", ctx.Project)
				assert.Equal(t, "test-stage", ctx.Stage)
				assert.NotNil(t, ctx.Config)
				assert.NotNil(t, ctx.CredentialsDB)
				assert.NotNil(t, ctx.KargoClient)
				assert.NotNil(t, ctx.ArgoCDClient)
			},
		},
		{
			name:        "context without permissions",
			step:        HealthCheckStep{},
			permissions: StepRunnerPermissions{},
			assertions: func(t *testing.T, ctx *HealthCheckStepContext) {
				assert.Nil(t, ctx.CredentialsDB)
				assert.Nil(t, ctx.KargoClient)
				assert.Nil(t, ctx.ArgoCDClient)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &SimpleEngine{
				credentialsDB: &credentials.FakeDB{},
				kargoClient:   fake.NewClientBuilder().Build(),
				argoCDClient:  fake.NewClientBuilder().Build(),
			}

			reg := HealthCheckStepRunnerRegistration{
				Permissions: tt.permissions,
			}

			ctx := engine.prepareHealthCheckStepContext(tt.healthCtx, tt.step, reg)
			tt.assertions(t, ctx)
		})
	}
}
