package promotion

import "context"

// mockStepRunner is a mock implementation of the StepRunner interface, which
// can be used for testing.
type mockStepRunner struct {
	// RunnerName is the RunnerName of the StepRunner.
	RunnerName string
	// RunFunc is the function that the StepRunner should call when Run is called.
	// If set, this function will be called instead of returning RunResult and
	// RunErr.
	RunFunc func(context.Context, *StepContext) (StepResult, error)
	// RunResult is the result that the StepRunner should return when Run is
	// called.
	RunResult StepResult
	// RunErr is the error that the StepRunner should return when Run is called.
	RunErr error
}

// Name implements the StepRunner interface.
func (m *mockStepRunner) Name() string {
	return m.RunnerName
}

// Run implements the promotion.StepRunner interface.
func (m *mockStepRunner) Run(ctx context.Context, stepCtx *StepContext) (StepResult, error) {
	if m.RunFunc != nil {
		return m.RunFunc(ctx, stepCtx)
	}
	return m.RunResult, m.RunErr
}
