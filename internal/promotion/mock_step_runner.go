package promotion

import (
	"context"
	"time"

	"github.com/akuity/kargo/pkg/promotion"
)

// mockStepRunner is a mock implementation of the StepRunner interface, which
// can be used for testing.
type mockStepRunner struct {
	// name is the name of the StepRunner.
	name string
	// runFunc is the function that the StepRunner should call when Run is called.
	// If set, this function will be called instead of returning RunResult and
	// RunErr.
	runFunc func(context.Context, *promotion.StepContext) (promotion.StepResult, error)
	// runResult is the result that the StepRunner should return when Run is
	// called.
	runResult promotion.StepResult
	// runErr is the error that the StepRunner should return when Run is called.
	runErr error
}

// Name implements the StepRunner interface.
func (m *mockStepRunner) Name() string {
	return m.name
}

// Run implements the promotion.StepRunner interface.
func (m *mockStepRunner) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, stepCtx)
	}
	return m.runResult, m.runErr
}

type MockRetryableStepRunner struct {
	*mockStepRunner
	defaultTimeout        *time.Duration
	defaultErrorThreshold uint32
}

func NewMockRetryableStepRunner(
	name string,
	defaultTimeout *time.Duration,
	defaultErrThreshold uint32,
) MockRetryableStepRunner {
	return MockRetryableStepRunner{
		mockStepRunner:        &mockStepRunner{name: name},
		defaultTimeout:        defaultTimeout,
		defaultErrorThreshold: defaultErrThreshold,
	}
}

func (m MockRetryableStepRunner) DefaultTimeout() *time.Duration {
	return m.defaultTimeout
}

func (m MockRetryableStepRunner) DefaultErrorThreshold() uint32 {
	return m.defaultErrorThreshold
}
