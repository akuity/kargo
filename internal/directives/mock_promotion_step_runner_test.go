package directives

import "context"

// mockPromotionStepRunner is a mock implementation of the PromotionStepRunner
// interface, which can be used for testing.
type mockPromotionStepRunner struct {
	// name is the name of the PromotionStepRunner.
	name string
	// runFunc is the function that the step should call when RunPromotionStep is
	// called. If set, this function will be called instead of returning runResult
	// and runErr.
	runFunc func(context.Context, *PromotionStepContext) (PromotionStepResult, error)
	// runResult is the result that the PromotionStepRunner should return when
	// RunPromotionStep is called.
	runResult PromotionStepResult
	// runErr is the error that the PromotionStepRunner should return when
	// RunPromotionStep is called.
	runErr error
}

// Name implements the PromotionStepRunner interface.
func (m *mockPromotionStepRunner) Name() string {
	return m.name
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (m *mockPromotionStepRunner) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, stepCtx)
	}
	return m.runResult, m.runErr
}
