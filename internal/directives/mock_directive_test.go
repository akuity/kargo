package directives

import "context"

// mockDirective is a mock implementation of the Directive interface, which can be
// used for testing.
type mockDirective struct {
	// name is the name of the Directive.
	name string
	// runFunc is the function that the step should call when RunPromotionStep is
	// called. If set, this function will be called instead of returning runResult
	// and runErr.
	runFunc func(context.Context, *PromotionStepContext) (PromotionStepResult, error)
	// runResult is the result that the Directive should return when
	// RunPromotionStep is called.
	runResult PromotionStepResult
	// runErr is the error that the Directive should return when RunPromotionStep
	// is called.
	runErr error
	// healthFunc is the function that the step should call when
	// RunHealthCheckStep is called. If set, this function will be called instead
	// of returning healthResult.
	healthFunc func(context.Context, *HealthCheckStepContext) HealthCheckStepResult
	// healthResult is the result that the Directive should return when
	// RunHealthCheckStep is called.
	healthResult HealthCheckStepResult
}

func (d *mockDirective) Name() string {
	return d.name
}

func (d *mockDirective) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	if d.runFunc != nil {
		return d.runFunc(ctx, stepCtx)
	}
	return d.runResult, d.runErr
}

func (d *mockDirective) RunHealthCheckStep(
	ctx context.Context,
	stepCtx *HealthCheckStepContext,
) HealthCheckStepResult {
	if d.healthFunc != nil {
		return d.healthFunc(ctx, stepCtx)
	}
	return d.healthResult
}
