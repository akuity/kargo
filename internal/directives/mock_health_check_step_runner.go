package directives

import "context"

// mockHealthChecker is a mock implementation of the HealthChecker interface,
// which can be used for testing.
type mockHealthChecker struct {
	// name is the name of the HealthChecker.
	name string
	// checkHealthFunc is the function that the step should call when CheckHealth
	// is called. If set, this function will be called instead of returning
	// checkHealthResult.
	checkHealthFunc func(context.Context, *HealthCheckStepContext) HealthCheckStepResult
	// checkHealthResult is the result that the HealthChecker should return when
	// CheckHealth is called.
	checkHealthResult HealthCheckStepResult
}

// Name implements the NamedRunner interface.
func (m *mockHealthChecker) Name() string {
	return m.name
}

// CheckHealth implements the HealthChecker interface.
func (m *mockHealthChecker) CheckHealth(
	ctx context.Context,
	stepCtx *HealthCheckStepContext,
) HealthCheckStepResult {
	if m.checkHealthFunc != nil {
		return m.checkHealthFunc(ctx, stepCtx)
	}
	return m.checkHealthResult
}
