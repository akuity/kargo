package promotion

import (
	"context"
	"fmt"
	"slices"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/health"
)

// StepRunner is an interface for components that implement the logic for
// execution of an individual Step in a user-defined promotion process.
type StepRunner interface {
	// Run executes an individual Step from a user-defined promotion process
	// using the provided StepContext. Implementations may indirectly modify
	// that context through the returned StepResult to allow StepRunners of
	// subsequent Steps to access the results of this execution.
	Run(context.Context, *StepContext) (StepResult, error)
}

// Context is the context of a user-defined promotion process that is executed
// by the Engine.
type Context struct {
	// UIBaseURL may be used to construct deeper URLs for interacting with the
	// Kargo UI.
	UIBaseURL string
	// WorkDir is the working directory to use for the Promotion.
	WorkDir string
	// Project is the Project that the Promotion is associated with.
	Project string
	// Stage is the Stage that the Promotion is targeting.
	Stage string
	// Promotion is the name of the Promotion.
	Promotion string
	// FreightRequests is the list of Freight from various origins that is
	// requested by the Stage targeted by the Promotion.
	//
	// This information is sometimes useful to Steps that reference a particular
	// artifact. When explicit information about the artifact's origin is absent,
	// Steps may need to examine FreightRequests.
	// This examination helps determine whether any ambiguity exists regarding
	// the artifact's origin. If ambiguity is found, a user may need to resolve
	// it.
	FreightRequests []kargoapi.FreightRequest
	// Freight is the collection of all Freight referenced by the Promotion. This
	// collection contains both the Freight that is actively being promoted and
	// any Freight that has been inherited from the target Stage's current
	// state.
	Freight kargoapi.FreightCollection
	// TargetFreightRef is the actual Freight that triggered this Promotion.
	TargetFreightRef kargoapi.FreightReference
	// StartFromStep is the index of the step from which the promotion should
	// begin execution.
	StartFromStep int64
	// StepExecutionMetadata tracks metadata pertaining to the execution
	// of individual promotion steps.
	StepExecutionMetadata kargoapi.StepExecutionMetadataList
	// State is the current state of the promotion process.
	State State
	// Vars is a list of variable definitions that can be used by the
	// Steps.
	Vars []kargoapi.ExpressionVariable
	// Actor is the name of the actor triggering the Promotion.
	Actor string

	// currentStepMetadata is a pointer to the StepMetadata for the
	// current step being executed. It is used to track the execution state of
	// the current step.
	currentStepMetadata *StepMetadata
}

// Result is the result of a user-defined promotion process executed by the
// Engine. It aggregates the status and output of the individual StepResults
// returned by the StepRunner executing each Step.
type Result struct {
	// Status is the high-level outcome of the user-defined promotion executed by
	// the Engine.
	Status kargoapi.PromotionPhase
	// Message is an optional message that provides additional context about the
	// outcome of the user-defined promotion executed by the Engine.
	Message string
	// HealthChecks collects health.Criteria returned from the execution of
	// individual Steps by their corresponding StepRunners. These criteria can
	// later be used as input to health.Checkers.
	HealthChecks []health.Criteria
	// If the promotion process remains in-progress, perhaps waiting for a change
	// in some external state, the value of this field will indicate where to
	// resume the process in the next reconciliation.
	CurrentStep int64
	// StepExecutionMetadata tracks metadata pertaining to the execution
	// of individual promotion steps.
	StepExecutionMetadata kargoapi.StepExecutionMetadataList
	// State is the current state of the promotion process.
	State State
	// RetryAfter is an optional, SUGGESTED duration after which a Promotion
	// reporting itself to be in a Running status should be retried. Note: This is
	// unrelated to retrying upon non-terminal failures.
	RetryAfter *time.Duration
}

// ContextOption is a function that configures a Context.
type ContextOption func(*Context)

// WithUIBaseURL sets the UIBaseURL of the Context.
func WithUIBaseURL(url string) ContextOption {
	return func(c *Context) {
		c.UIBaseURL = url
	}
}

// WithWorkDir sets the WorkDir of the Context.
func WithWorkDir(dir string) ContextOption {
	return func(c *Context) {
		c.WorkDir = dir
	}
}

// WithActor sets the Actor of the Context.
func WithActor(actor string) ContextOption {
	return func(c *Context) {
		c.Actor = actor
	}
}

// NewContext creates a new Context for a user-defined promotion process
// executed by the Engine. It initializes the Context with the data from the
// provided Promotion and Stage, applying any additional options provided.
func NewContext(
	promo *kargoapi.Promotion,
	stage *kargoapi.Stage,
	opts ...ContextOption,
) Context {
	ctx := Context{
		Project:               promo.Namespace,
		Stage:                 promo.Spec.Stage,
		Promotion:             promo.Name,
		StartFromStep:         promo.Status.CurrentStep,
		StepExecutionMetadata: promo.Status.StepExecutionMetadata,
		State:                 State(promo.Status.GetState()),
		Vars:                  promo.Spec.Vars,
	}

	if stage != nil && len(stage.Spec.RequestedFreight) > 0 {
		ctx.FreightRequests = make([]kargoapi.FreightRequest, len(stage.Spec.RequestedFreight))
		for i, fr := range stage.Spec.RequestedFreight {
			ctx.FreightRequests[i] = *fr.DeepCopy()
		}
	}

	if promo.Status.Freight != nil {
		ctx.TargetFreightRef = *promo.Status.Freight.DeepCopy()
	}

	if promo.Status.FreightCollection != nil {
		ctx.Freight = *promo.Status.FreightCollection.DeepCopy()
	}

	for _, opt := range opts {
		opt(&ctx)
	}

	return ctx
}

// SetCurrentStep sets the current step to the provided Step and returns a
// StepMetadata that can be used to update the execution metadata of the step.
func (c *Context) SetCurrentStep(step Step) *StepMetadata {
	meta := c.GetStepExecutionMetadata(step)
	c.currentStepMetadata = (*StepMetadata)(meta)
	return c.currentStepMetadata
}

// GetCurrentStep retrieves the StepMetadata for the current step being executed.
// If no current step is set, it returns nil.
func (c *Context) GetCurrentStep() *StepMetadata {
	return c.currentStepMetadata
}

// GetCurrentStepIndex retrieves the index of the current step being executed.
// It derives this from the length of StepExecutionMetadata, which is built
// incrementally as steps are encountered.
func (c *Context) GetCurrentStepIndex() int64 {
	if len(c.StepExecutionMetadata) == 0 {
		return 0
	}
	return int64(len(c.StepExecutionMetadata) - 1)
}

// GetStepExecutionMetadata retrieves the StepExecutionMetadata for a given
// Step. If metadata for the Step does not already exist, it creates a new
// StepExecutionMetadata entry with the Step's alias and ContinueOnError
// property, and returns it.
func (c *Context) GetStepExecutionMetadata(step Step) *kargoapi.StepExecutionMetadata {
	for i := range c.StepExecutionMetadata {
		if c.StepExecutionMetadata[i].Alias == step.Alias {
			// Found existing metadata for this step, return it.
			return &c.StepExecutionMetadata[i]
		}
	}

	// If not found, append new metadata
	c.StepExecutionMetadata = append(
		c.StepExecutionMetadata,
		kargoapi.StepExecutionMetadata{
			Alias:           step.Alias,
			ContinueOnError: step.ContinueOnError,
		},
	)
	return &c.StepExecutionMetadata[len(c.StepExecutionMetadata)-1]
}

// DeepCopy creates a deep copy of the Context. It can be used to ensure that
// modifications to the Context do not affect the original Context.
func (c *Context) DeepCopy() Context {
	newC := Context{
		UIBaseURL:             c.UIBaseURL,
		WorkDir:               c.WorkDir,
		Project:               c.Project,
		Stage:                 c.Stage,
		Promotion:             c.Promotion,
		Freight:               *c.Freight.DeepCopy(),
		TargetFreightRef:      *c.TargetFreightRef.DeepCopy(),
		StartFromStep:         c.StartFromStep,
		StepExecutionMetadata: c.StepExecutionMetadata.DeepCopy(),
		State:                 c.State.DeepCopy(),
		Vars:                  slices.Clone(c.Vars),
		Actor:                 c.Actor,
	}

	if c.FreightRequests != nil {
		newC.FreightRequests = make([]kargoapi.FreightRequest, len(c.FreightRequests))
		for i, fr := range c.FreightRequests {
			newC.FreightRequests[i] = *fr.DeepCopy()
		}
	}

	return newC
}

// Step describes a single step in a user-defined promotion process. Steps are
// executed in sequence by the Engine, which delegates of each to a StepRunner.
type Step struct {
	// Kind identifies a registered StepRunner that implements the logic for this
	// step of the user-defined promotion process.
	Kind string
	// Alias is an optional identifier for this step of the use-defined promotion
	// process, which must be unique to the process. Output from execution of the
	// step will be keyed to this alias by the Engine and made accessible to
	// subsequent steps.
	Alias string
	// If is an optional expression that, if present, must evaluate to a boolean
	// value. If the expression evaluates to false, the step will be skipped.
	// If the expression does not evaluate to a boolean value, the step will
	// fail.
	If string
	// ContinueOnError is a boolean value that, if set to true, will cause the
	// Promotion to continue executing the next step even if this step fails. It
	// also will not permit this failure to impact the overall status of the
	// Promotion.
	ContinueOnError bool
	// Retry is the retry configuration for the Step.
	Retry *kargoapi.PromotionStepRetry
	// Vars is a list of variables definitions that can be used by the
	// Step.
	Vars []kargoapi.ExpressionVariable
	// Config is an opaque JSON to be passed to the StepRunner executing this
	// step.
	Config []byte
}

// NewSteps creates a slice of Steps from the provided Promotion. Each Step in
// the slice corresponds to a step defined in the Promotion's spec.
func NewSteps(promo *kargoapi.Promotion) []Step {
	result := make([]Step, len(promo.Spec.Steps))
	for i, step := range promo.Spec.Steps {
		var rawConfig []byte
		if step.Config != nil {
			rawConfig = step.Config.Raw
		}
		result[i] = Step{
			Kind:            step.Uses,
			Alias:           step.As,
			If:              step.If,
			ContinueOnError: step.ContinueOnError,
			Retry:           step.Retry,
			Vars:            step.Vars,
			Config:          rawConfig,
		}
	}
	return result
}

// StepMetadata is a type that represents metadata about the execution of a
// single step in a user-defined promotion process. It is used to track the
// status, start and finish times, error counts, and other relevant information
// about the step's execution. This metadata is stored in the
// StepExecutionMetadata field of the Context and is used to provide detailed
// information about the execution of each step in the promotion process.
type StepMetadata kargoapi.StepExecutionMetadata

// WithStatus sets the status of the StepMetadata and returns the updated
// StepMetadata. This method is used to update the status of the step during
// its execution, such as when it starts, finishes, or encounters an error.
func (m *StepMetadata) WithStatus(status kargoapi.PromotionStepStatus) *StepMetadata {
	m.Status = status
	return m
}

// WithMessage sets the message of the StepMetadata and returns the updated
// StepMetadata. This method is used to provide additional context or details
// about the step's execution, such as error messages or informational messages
// that may be useful for debugging or understanding the step's outcome.
func (m *StepMetadata) WithMessage(message string) *StepMetadata {
	m.Message = message
	return m
}

// WithMessagef formats the message using the provided format string and
// arguments, sets it as the message of the StepMetadata, and returns the
// updated StepMetadata. This method is useful for constructing dynamic messages
// that include variable content, such as error details or step-specific
// information.
func (m *StepMetadata) WithMessagef(format string, a ...any) *StepMetadata {
	m.Message = fmt.Sprintf(format, a...)
	return m
}

// Error increments the error count of the StepMetadata and returns the updated
// StepMetadata. This method is used to track the number of errors encountered
// during the execution of the step. It is typically called when the step fails
// or encounters an error condition.
func (m *StepMetadata) Error() *StepMetadata {
	m.ErrorCount++
	return m
}

// Started sets the StartedAt timestamp to the current time if it is not already
// set, and resets the error count to zero. It returns the updated StepMetadata.
// This method is used to mark the start of the step's execution, indicating
// when the step began processing.
func (m *StepMetadata) Started() *StepMetadata {
	if m.StartedAt == nil {
		m.StartedAt = ptr.To(metav1.Now())
		m.ErrorCount = 0
	}
	return m
}

// Finished sets the FinishedAt timestamp to the current time if it is not
// already set, indicating that the step has completed its execution. It returns
// the updated StepMetadata. This method is used to mark the end of the step's
// execution, indicating when the step finished processing.
func (m *StepMetadata) Finished() *StepMetadata {
	if m.FinishedAt == nil {
		m.FinishedAt = ptr.To(metav1.Now())
	}
	return m
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
	// RetryAfter is an optional, SUGGESTED duration after which a step reporting
	// itself to be in a Running status should be retried. Note: This is unrelated
	// to retrying upon non-terminal failures.
	RetryAfter *time.Duration
}
