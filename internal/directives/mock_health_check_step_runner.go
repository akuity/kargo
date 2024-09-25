package directives

import "context"

// mockHealthCheckStepRunner is a mock implementation of the
// HealthCheckStepRunner interface, which can be used for testing.
type mockHealthCheckStepRunner struct {
	// name is the name of the HealthCheckStepRunner.
	name string
	// runFunc is the function that the step should call when
	// RunHealthCheckStep is called. If set, this function will be called instead
	// of returning healthResult.
	runFunc func(context.Context, *HealthCheckStepContext) HealthCheckStepResult
	// runResult is the result that the HealthCheckStepRunner should return when
	// RunHealthCheckStep is called.
	runResult HealthCheckStepResult
}

// Name implements the HealthCheckStepRunner interface.
func (m *mockHealthCheckStepRunner) Name() string {
	return m.name
}

// RunHealthCheckStep implements the HealthCheckStepRunner interface.
func (m *mockHealthCheckStepRunner) RunHealthCheckStep(
	ctx context.Context,
	stepCtx *HealthCheckStepContext,
) HealthCheckStepResult {
	if m.runFunc != nil {
		return m.runFunc(ctx, stepCtx)
	}
	return m.runResult
}
