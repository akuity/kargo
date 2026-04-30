package promotion

import (
	"context"
)

// StepExecutionRequest represents a request to execute a Step. It contains
// all the necessary Context and Step information required for executing the
// Step.
type StepExecutionRequest struct {
	Context StepContext `json:"context"`
	Step    Step        `json:"step"`
}

// StepExecutor defines the interface for executing a single Step.
type StepExecutor interface {
	ExecuteStep(ctx context.Context, req StepExecutionRequest) (StepResult, error)
}
