package promotion

import "context"

// MockStepRunner is a mock implementation of the StepRunner interface, which
// can be used for testing.
type MockStepRunner struct {
	// RunFunc is the function that this MockStepRunner should call when Run is
	// called. If set, this function will be called instead of returning RunResult
	// and RunErr.
	RunFunc func(context.Context, *StepContext) (StepResult, error)
	// RunResult is the result that this MockStepRunner should return when Run is
	// called.
	RunResult StepResult
	// RunErr is the error that this MockStepRunner should return when Run is
	// called.
	RunErr error
}

// Run implements the StepRunner interface.
func (m *MockStepRunner) Run(
	ctx context.Context,
	stepCtx *StepContext,
) (StepResult, error) {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, stepCtx)
	}
	return m.RunResult, m.RunErr
}
