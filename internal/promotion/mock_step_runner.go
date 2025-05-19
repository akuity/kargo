package promotion

import (
	"time"

	"github.com/akuity/kargo/pkg/promotion"
)

type MockRetryableStepRunner struct {
	*promotion.MockStepRunner
	defaultTimeout        *time.Duration
	defaultErrorThreshold uint32
}

func NewMockRetryableStepRunner(
	name string,
	defaultTimeout *time.Duration,
	defaultErrThreshold uint32,
) MockRetryableStepRunner {
	return MockRetryableStepRunner{
		MockStepRunner:        &promotion.MockStepRunner{Nm: name},
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
