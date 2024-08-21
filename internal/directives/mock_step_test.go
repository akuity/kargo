package directives

import "context"

// mockStep is a mock implementation of the Step interface, which can be
// used for testing.
type mockStep struct {
	// name is the name of the step.
	name string
	// runFunc is the function that the step should call when Run is called.
	// If set, this function will be called instead of returning runResult
	// and runErr.
	runFunc func(context.Context, *StepContext) (Result, error)
	// runResult is the result that the step should return when Run is called.
	runResult Result
	// runErr is the error that the step should return when Run is called.
	runErr error
}

func (d *mockStep) Name() string {
	return d.name
}

func (d *mockStep) Run(ctx context.Context, stepCtx *StepContext) (Result, error) {
	if d.runFunc != nil {
		return d.runFunc(ctx, stepCtx)
	}
	return d.runResult, d.runErr
}
