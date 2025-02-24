package directives

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	yaml "sigs.k8s.io/yaml/goyaml.v3"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/expressions"
	exprfn "github.com/akuity/kargo/internal/expressions/function"
	"github.com/akuity/kargo/internal/kargo"
)

// PromotionStepRunner is an interface for components that implement the logic for
// execution of the individual PromotionSteps of a user-defined promotion
// process.
type PromotionStepRunner interface {
	// Name returns the name of the PromotionStepRunner.
	Name() string
	// RunPromotionStep executes an individual step of a user-defined promotion
	// process using the provided PromotionStepContext. Implementations may
	// indirectly modify that context through the returned PromotionStepResult to
	// allow subsequent promotion steps to access the results of its execution.
	RunPromotionStep(context.Context, *PromotionStepContext) (PromotionStepResult, error)
}

// RetryableStepRunner is an interface for PromotionStepRunners that can be
// retried in the event of a failure.
type RetryableStepRunner interface {
	// DefaultTimeout returns the default timeout for the step.
	DefaultTimeout() *time.Duration
	// DefaultErrorThreshold returns the number of consecutive times the step must
	// fail (for any reason) before retries are abandoned and the entire Promotion
	// is marked as failed.
	DefaultErrorThreshold() uint32
}

// PromotionContext is the context of a user-defined promotion process that is
// executed by the Engine.
type PromotionContext struct {
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
	// requested by the Stage targeted by the Promotion. This information is
	// sometimes useful to PromotionSteps that reference a particular artifact
	// and, in the absence of any explicit information about the origin of that
	// artifact, may need to examine FreightRequests to determine whether there
	// exists any ambiguity as to its origin, which a user may then need to
	// resolve.
	FreightRequests []kargoapi.FreightRequest
	// Freight is the collection of all Freight referenced by the Promotion. This
	// collection contains both the Freight that is actively being promoted and
	// any Freight that has been inherited from the target Stage's current
	// state.
	Freight kargoapi.FreightCollection
	// StartFromStep is the index of the step from which the promotion should
	// begin execution.
	StartFromStep int64
	// StepExecutionMetadata tracks metadata pertaining to the execution
	// of individual promotion steps.
	StepExecutionMetadata kargoapi.StepExecutionMetadataList
	// State is the current state of the promotion process.
	State State
	// Vars is a list of variables definitions that can be used by the
	// PromotionSteps.
	Vars []kargoapi.PromotionVariable
	// Secrets is a map of secrets that can be used by the PromotionSteps.
	Secrets map[string]map[string]string
}

// PromotionStep describes a single step in a user-defined promotion process.
// PromotionSteps are executed in sequence by the Engine, which delegates the
// execution of each step to a PromotionStepRunner.
type PromotionStep struct {
	// Kind identifies a registered PromotionStepRunner that implements the logic
	// for this step of the user-defined promotion process.
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
	// Retry is the retry configuration for the PromotionStep.
	Retry *kargoapi.PromotionStepRetry
	// Vars is a list of variables definitions that can be used by the
	// PromotionStep.
	Vars []kargoapi.PromotionVariable
	// Config is an opaque JSON to be passed to the PromotionStepRunner executing
	// this step.
	Config []byte
}

// PromotionStepEnvOption is a functional option for customizing the
// environment of a PromotionStep built by BuildEnv.
type PromotionStepEnvOption func(map[string]any)

// StepEnvWithVars returns a PromotionStepEnvOption that adds the provided vars to
// the environment of the PromotionStep.
func StepEnvWithVars(vars map[string]any) PromotionStepEnvOption {
	return func(env map[string]any) {
		env["vars"] = vars
	}
}

// StepEnvWithSecrets returns a PromotionStepEnvOption that adds the provided secrets
// to the environment of the PromotionStep.
func StepEnvWithSecrets(secrets map[string]map[string]string) PromotionStepEnvOption {
	return func(env map[string]any) {
		env["secrets"] = secrets
	}
}

// StepEnvWithOutputs returns a PromotionStepEnvOption that adds the provided outputs
// to the environment of the PromotionStep.
func StepEnvWithOutputs(outputs State) PromotionStepEnvOption {
	return func(env map[string]any) {
		env["outputs"] = outputs
	}
}

// StepEnvWithTaskOutputs returns a PromotionStepEnvOption that adds the provided
// task outputs to the environment of the PromotionStep.
func StepEnvWithTaskOutputs(alias string, outputs State) PromotionStepEnvOption {
	return func(env map[string]any) {
		// Ensure that if the PromotionStep originated from a task, the task outputs
		// are available to the PromotionStep. This allows inflated steps to access
		// the outputs of the other steps in the task without needing to know the
		// alias (namespace) of the task.
		if taskOutput := getTaskOutputs(alias, outputs); taskOutput != nil {
			env["task"] = map[string]any{
				"outputs": taskOutput,
			}
		}
	}
}

// GetTimeout returns the maximum interval the provided runner may spend
// attempting to execute the step before retries are abandoned and the entire
// Promotion is marked as failed. If the runner is a RetryableStepRunner, its
// timeout is used as the default. Otherwise, the default is 0 (no limit).
func (s *PromotionStep) GetTimeout(runner any) *time.Duration {
	fallback := ptr.To(time.Duration(0))
	if retryCfg, isRetryable := runner.(RetryableStepRunner); isRetryable {
		fallback = retryCfg.DefaultTimeout()
	}
	return s.Retry.GetTimeout(fallback)
}

// GetErrorThreshold returns the number of consecutive times the provided runner
// must fail to execute the step (for any reason) before retries are abandoned
// and the entire Promotion is marked as failed. If the runner is a
// RetryableStepRunner, its threshold is used as the default. Otherwise, the
// default is 1.
func (s *PromotionStep) GetErrorThreshold(runner any) uint32 {
	fallback := uint32(1)
	if retryCfg, isRetryable := runner.(RetryableStepRunner); isRetryable {
		fallback = retryCfg.DefaultErrorThreshold()
	}
	return s.Retry.GetErrorThreshold(fallback)
}

// BuildEnv returns the environment for the PromotionStep. The environment
// includes the context of the Promotion and any additional options provided
// (e.g. outputs, task outputs, vars, secrets).
//
// The environment is a (nested) map of string keys to any values. The keys
// are used as variables in the PromotionStep configuration.
func (s *PromotionStep) BuildEnv(
	promoCtx PromotionContext,
	opts ...PromotionStepEnvOption,
) map[string]any {
	env := map[string]any{
		"ctx": map[string]any{
			"project":   promoCtx.Project,
			"promotion": promoCtx.Promotion,
			"stage":     promoCtx.Stage,
		},
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(env)
	}

	return env
}

// Skip returns true if the PromotionStep should be skipped based on the If
// condition. The If condition is evaluated in the context of the provided
// PromotionContext and State.
func (s *PromotionStep) Skip(
	promoCtx PromotionContext,
	state State,
) (bool, error) {
	if s.If == "" {
		return false, nil
	}

	vars, err := s.GetVars(promoCtx, state)
	if err != nil {
		return false, err
	}

	env := s.BuildEnv(
		promoCtx,
		StepEnvWithOutputs(state),
		StepEnvWithTaskOutputs(s.Alias, state),
		StepEnvWithVars(vars),
	)

	v, err := expressions.EvaluateTemplate(s.If, env)
	if err != nil {
		return false, err
	}

	if b, ok := v.(bool); ok {
		return !b, nil
	}

	return false, fmt.Errorf("expression must evaluate to a boolean")
}

// GetConfig returns the Config unmarshalled into a map. Any expr-lang
// expressions are evaluated in the context of the provided arguments
// prior to unmarshaling.
func (s *PromotionStep) GetConfig(
	ctx context.Context,
	cl client.Client,
	promoCtx PromotionContext,
	state State,
) (Config, error) {
	if s.Config == nil {
		return nil, nil
	}

	vars, err := s.GetVars(promoCtx, state)
	if err != nil {
		return nil, err
	}

	env := s.BuildEnv(
		promoCtx,
		StepEnvWithOutputs(state),
		StepEnvWithTaskOutputs(s.Alias, state),
		StepEnvWithVars(vars),
		StepEnvWithSecrets(promoCtx.Secrets),
	)

	evaledCfgJSON, err := expressions.EvaluateJSONTemplate(
		s.Config,
		env,
		exprfn.FreightOperations(
			ctx,
			cl,
			promoCtx.Project,
			promoCtx.FreightRequests,
			promoCtx.Freight.References(),
		)...,
	)
	if err != nil {
		return nil, err
	}
	var config map[string]any
	if err := yaml.Unmarshal(evaledCfgJSON, &config); err != nil {
		return nil, nil
	}
	return config, nil
}

// GetVars returns the variables defined in the PromotionStep. The variables are
// evaluated in the context of the provided PromotionContext.
func (s *PromotionStep) GetVars(
	promoCtx PromotionContext,
	state State,
) (map[string]any, error) {
	vars := make(map[string]any)

	// Evaluate the global variables defined in the Promotion itself, these
	// variables DO NOT have access to the (task) outputs.
	for _, v := range promoCtx.Vars {
		newVar, err := expressions.EvaluateTemplate(
			v.Value,
			s.BuildEnv(promoCtx, StepEnvWithVars(vars)),
		)
		if err != nil {
			return nil, fmt.Errorf("error pre-processing promotion variable %q: %w", v.Name, err)
		}
		vars[v.Name] = newVar
	}

	// Evaluate the variables defined in the PromotionStep, these variables
	// DO have access to the (task) outputs.
	for _, v := range s.Vars {
		newVar, err := expressions.EvaluateTemplate(
			v.Value,
			s.BuildEnv(
				promoCtx,
				StepEnvWithOutputs(state),
				StepEnvWithTaskOutputs(s.Alias, state),
				StepEnvWithVars(vars),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("error pre-processing promotion variable %q: %w", v.Name, err)
		}
		vars[v.Name] = newVar
	}

	return vars, nil
}

// PromotionResult is the result of a user-defined promotion process executed by
// the Engine. It aggregates the status and output of the individual
// PromotionStepResults returned by the PromotionStepRunner executing each
// PromotionStep.
type PromotionResult struct {
	// Status is the high-level outcome of the user-defined promotion executed by
	// the Engine.
	Status kargoapi.PromotionPhase
	// Message is an optional message that provides additional context about the
	// outcome of the user-defined promotion executed by the Engine.
	Message string
	// HealthCheckSteps collects health check configuration returned from the
	// execution of individual PromotionSteps by their corresponding
	// PromotionStepRunners. This configuration can later be used as input to
	// health check processes.
	HealthCheckSteps []HealthCheckStep
	// If the promotion process remains in-progress, perhaps waiting for a change
	// in some external state, the value of this field will indicate where to
	// resume the process in the next reconciliation.
	CurrentStep int64
	// StepExecutionMetadata tracks metadata pertaining to the execution
	// of individual promotion steps.
	StepExecutionMetadata kargoapi.StepExecutionMetadataList
	// State is the current state of the promotion process.
	State State
}

// PromotionStepContext is a type that represents the context in which a
// SinglePromotion step is executed by a PromotionStepRunner.
type PromotionStepContext struct {
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
	// FreightRequests is the list of Freight from various origins that is
	// requested by the Stage targeted by the Promotion. This information is
	// sometimes useful to PromotionStep that reference a particular artifact and,
	// in the absence of any explicit information about the origin of that
	// artifact, may need to examine FreightRequests to determine whether there
	// exists any ambiguity as to its origin, which a user may then need to
	// resolve.
	//
	// TODO: krancour: Longer term, if we can standardize the way that
	// PromotionSteps express the artifacts they need to work with, we can make
	// the Engine responsible for finding them and furnishing them directly to
	// each PromotionStepRunner.
	FreightRequests []kargoapi.FreightRequest
	// Freight is the collection of all Freight referenced by the Promotion. This
	// collection contains both the Freight that is actively being promoted as
	// well as any Freight that has been inherited from the target Stage's current
	// state.
	//
	// TODO: krancour: Longer term, if we can standardize the way that
	// PromotionSteps express the artifacts they need to work with, we can make
	// the Engine responsible for finding them and furnishing them directly to
	// each PromotionStepRunner.
	Freight kargoapi.FreightCollection
	// KargoClient is a Kubernetes client that a PromotionStepRunner executing a
	// PromotionStep may use to interact with the Kargo control plane. The value
	// of this field will often be nil, as the Engine will only furnish a this to
	// specially privileged PromotionStepRunners.
	//
	// TODO: krancour: Longer term, we may be able to do without this. See notes
	// on previous two fields.
	KargoClient client.Client
	// ArgoCDClient is a Kubernetes client that a PromotionStepRunner executing a
	// PromotionStep may use to interact with an Argo CD control plane. The value
	// of this field will often be nil, as the Engine will only furnish this to
	// specially privileged PromotionStepRunners.
	ArgoCDClient client.Client
	// CredentialsDB is a database of credentials that a PromotionStepRunner
	// executing a PromotionStep may use to acquire credentials for interacting
	// with external systems. The value of this field will often be nil, as the
	// Engine will only furnish a CredentialsDB to specially privileged
	// PromotionStepRunners.
	//
	// TODO: krancour: Longer term, if we can standardize the way that
	// PromotionSteps express what credentials they need, we can make the Engine
	// responsible for finding them and furnishing them directly to each
	// PromotionStepRunner.
	CredentialsDB credentials.Database
}

// PromotionStepResult represents the results of single PromotionStep executed
// by a PromotionStepRunner.
type PromotionStepResult struct {
	// Status is the high-level outcome a PromotionStep executed by a
	// PromotionStepRunner.
	Status kargoapi.PromotionPhase
	// Message is an optional message that provides additional context about the
	// outcome of a PromotionStep executed by a PromotionStepRunner.
	Message string
	// Output is the opaque output of a PromotionStep executed by a
	// PromotionStepRunner. The Engine will update shared state with this output,
	// making it available to subsequent steps.
	Output map[string]any
	// HealthCheckStep is health check opaque configuration optionally returned by
	// a PromotionStepRunner's successful execution of a PromotionStep. This
	// configuration can later be used as input to health check processes.
	HealthCheckStep *HealthCheckStep
}

// getTaskOutputs returns the outputs of a task that are relevant to the current
// step. This is useful when a step is inflated from a task and needs to access
// the outputs of that task.
func getTaskOutputs(alias string, state State) State {
	if namespace := getAliasNamespace(alias); namespace != "" {
		taskOutputs := make(State)
		for k, v := range state.DeepCopy() {
			if getAliasNamespace(k) == namespace {
				taskOutputs[k[len(namespace)+2:]] = v
			}
		}
		return taskOutputs
	}
	return nil
}

// getAliasNamespace returns the namespace part of an alias, if it exists.
// The namespace part is the part before the first "::" separator. Typically,
// this is used for steps inflated from a task.
func getAliasNamespace(alias string) string {
	parts := strings.Split(alias, kargo.PromotionAliasSeparator)
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}
