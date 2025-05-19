package builtin

import (
	"context"
	"time"

	"github.com/akuity/kargo/pkg/promotion"
)

// retryableStepRunner is a wrapper around a promotion.StepRunner that
// implements promotion.RetryableStepRunner.
type retryableStepRunner struct {
	runner                promotion.StepRunner
	defaultTimeout        *time.Duration
	defaultErrorThreshold uint32
}

// NewRetryableStepRunner returns a wrapper around a promotion.StepRunner that
// implements promotion.RetryableStepRunner.
func NewRetryableStepRunner(
	runner promotion.StepRunner,
	timeout *time.Duration,
	errorThreshold uint32,
) promotion.RetryableStepRunner {
	return &retryableStepRunner{
		runner:                runner,
		defaultTimeout:        timeout,
		defaultErrorThreshold: errorThreshold,
	}
}

// Name implements promotion.StepRunner.
func (r *retryableStepRunner) Name() string {
	return r.runner.Name()
}

// Run implements promotion.StepRunner.
func (r *retryableStepRunner) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	return r.runner.Run(ctx, stepCtx)
}

// DefaultTimeout implements promotion.RetryableStepRunner.
func (r *retryableStepRunner) DefaultTimeout() *time.Duration {
	return r.defaultTimeout
}

// DefaultErrorThreshold implements promotion.RetryableStepRunner.
func (r *retryableStepRunner) DefaultErrorThreshold() uint32 {
	return r.defaultErrorThreshold
}
