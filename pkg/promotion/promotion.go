package promotion

import (
	"context"
	"time"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/health"
)

// StepRunner is an interface for components that implement the logic for
// execution of an individual Step in a user-defined promotion process.
type StepRunner interface {
	// Name returns the name of the StepRunner.
	Name() string
	// Run executes an individual Step from a user-defined promotion process using
	// the provided StepContext. Implementations may indirectly modify that
	// context through the returned StepResult to allow StepRunners for subsequent
	// Steps to access the results of this execution.
	Run(context.Context, *StepContext) (StepResult, error)
}

// RetryableStepRunner is an additional interface for StepRunner implementations
// that can be retried in the event of a failure.
type RetryableStepRunner interface {
	StepRunner
	// DefaultTimeout returns the default timeout for the step.
	DefaultTimeout() *time.Duration
	// DefaultErrorThreshold returns the number of consecutive times the step must
	// fail (for any reason) before retries are abandoned and the entire Promotion
	// is marked as failed.
	DefaultErrorThreshold() uint32
}

// retryableStepRunner is a wrapper around a StepRunner that implements
// RetryableStepRunner.
type retryableStepRunner struct {
	runner                StepRunner
	defaultTimeout        *time.Duration
	defaultErrorThreshold uint32
}

// NewRetryableStepRunner returns a wrapper around a StepRunner that implements
// RetryableStepRunner.
func NewRetryableStepRunner(
	runner StepRunner,
	timeout *time.Duration,
	errorThreshold uint32,
) RetryableStepRunner {
	return &retryableStepRunner{
		runner:                runner,
		defaultTimeout:        timeout,
		defaultErrorThreshold: errorThreshold,
	}
}

// Name implements RetryableStepRunner.
func (r *retryableStepRunner) Name() string {
	return r.runner.Name()
}

// Run implements RetryableStepRunner.
func (r *retryableStepRunner) Run(
	ctx context.Context,
	stepCtx *StepContext,
) (StepResult, error) {
	return r.runner.Run(ctx, stepCtx)
}

// DefaultTimeout implements RetryableStepRunner.
func (r *retryableStepRunner) DefaultTimeout() *time.Duration {
	return r.defaultTimeout
}

// DefaultErrorThreshold implements RetryableStepRunner.
func (r *retryableStepRunner) DefaultErrorThreshold() uint32 {
	return r.defaultErrorThreshold
}

// TaskLevelOutputStepRunner is an interface that defines a method to instruct
// the engine to propagate the output of a step to the task namespace in the
// shared state of the promotion result.
type TaskLevelOutputStepRunner interface {
	StepRunner
	// TaskLevelOutput returns true if the StepRunner produces output that
	// should be propagated to the task namespace in the state of the promotion
	// result by the engine after the step is executed.
	TaskLevelOutput() bool
}

// NewTaskLevelOutputStepRunner returns a wrapper around a StepRunner that
// implements TaskLevelOutputStepRunner.
func NewTaskLevelOutputStepRunner(runner StepRunner) TaskLevelOutputStepRunner {
	return &taskLevelOutputStepRunner{
		runner: runner,
	}
}

// taskLevelOutputStepRunner is a wrapper around a StepRunner that implements
// TaskLevelOutputStepRunner.
type taskLevelOutputStepRunner struct {
	runner StepRunner
}

// Name implements StepRunner.
func (r *taskLevelOutputStepRunner) Name() string {
	return r.runner.Name()
}

// Run implements StepRunner.
func (r *taskLevelOutputStepRunner) Run(
	ctx context.Context,
	stepCtx *StepContext,
) (StepResult, error) {
	return r.runner.Run(ctx, stepCtx)
}

// TaskLevelOutput implements TaskLevelOutputStepRunner.
func (r *taskLevelOutputStepRunner) TaskLevelOutput() bool {
	return true
}

// StepContext is a type that represents the context in which a
// single promotion step is executed by a StepRunner.
type StepContext struct {
	// UIBaseURL may be used to construct deeper URLs for interacting with the
	// Kargo UI.
	UIBaseURL string
	// WorkDir is the root directory for the execution of a step.
	WorkDir string
	// SharedState is the state shared between steps.
	SharedState State
	// Alias is the alias of the step that is currently being executed.
	Alias string
	// Config is the configuration of the step that is currently being
	// executed.
	Config Config
	// Project is the Project that the Promotion is associated with.
	Project string
	// Stage is the Stage that the Promotion is targeting.
	Stage string
	// Promotion is the name of the Promotion.
	Promotion string
	// PromotionActor is the name of the actor triggering the Promotion.
	PromotionActor string
	// FreightRequests is the list of Freight from various origins that is
	// requested by the Stage targeted by the Promotion. This information is
	// sometimes useful to Step that reference a particular artifact and, in the
	// absence of any explicit information about the origin of that artifact, may
	// need to examine FreightRequests to determine whether there exists any
	// ambiguity as to its origin, which a user may then need to resolve.
	//
	// TODO: krancour: Longer term, if we can standardize the way that Steps
	// express the artifacts they need to work with, we can make the Engine
	// responsible for finding them and furnishing them directly to each
	// StepRunner.
	FreightRequests []kargoapi.FreightRequest
	// Freight is the collection of all Freight referenced by the Promotion. This
	// collection contains both the Freight that is actively being promoted as
	// well as any Freight that has been inherited from the target Stage's current
	// state.
	//
	// TODO: krancour: Longer term, if we can standardize the way that Steps
	// express the artifacts they need to work with, we can make the Engine
	// responsible for finding them and furnishing them directly to each
	// StepRunner.
	Freight kargoapi.FreightCollection
	// TargetFreightRef is the actual Freight that triggered this Promotion.
	TargetFreightRef kargoapi.FreightReference
}

// StepResult represents the results of a single Step of a user-defined promotion
// process executed by a StepRunner.
type StepResult struct {
	// Status is the high-level outcome of a Step executed by a StepRunner.
	Status kargoapi.PromotionStepStatus
	// Message is an optional message that provides additional context about the
	// outcome of a Step executed by a StepRunner.
	Message string
	// Output is the opaque output of a Step executed by a StepRunner. The Engine
	// will update the shared state with this output, making it available to the
	// StepRunners executing subsequent Steps.
	Output map[string]any
	// HealthCheck identifies criteria for a health check process. This is
	// returned by some StepRunner upon successful execution of a Step. These
	// criteria can be used later as input to a health.Checker.
	HealthCheck *health.Criteria
}
