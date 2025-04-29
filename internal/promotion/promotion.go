package promotion

import (
	"context"
	"fmt"
	"strings"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	yaml "sigs.k8s.io/yaml/goyaml.v3"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/expressions"
	exprfn "github.com/akuity/kargo/internal/expressions/function"
	"github.com/akuity/kargo/internal/kargo"
	"github.com/akuity/kargo/pkg/health"
	"github.com/akuity/kargo/pkg/promotion"
)

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
	// requested by the Stage targeted by the Promotion. This information is
	// sometimes useful to Steps that reference a particular artifact and, in the
	// absence of any explicit information about the origin of that artifact, may
	// need to examine FreightRequests to determine whether there exists any
	// ambiguity as to its origin, which a user may then need to resolve.
	FreightRequests []kargoapi.FreightRequest
	// Freight is the collection of all Freight referenced by the Promotion. This
	// collection contains both the Freight that is actively being promoted and
	// any Freight that has been inherited from the target Stage's current
	// state.
	Freight kargoapi.FreightCollection
	// TargetFreightRef is the actual freight that triggered this Promotion.
	TargetFreightRef kargoapi.FreightReference
	// StartFromStep is the index of the step from which the promotion should
	// begin execution.
	StartFromStep int64
	// StepExecutionMetadata tracks metadata pertaining to the execution
	// of individual promotion steps.
	StepExecutionMetadata kargoapi.StepExecutionMetadataList
	// State is the current state of the promotion process.
	State promotion.State
	// Vars is a list of variables definitions that can be used by the
	// Steps.
	Vars []kargoapi.ExpressionVariable
	// Secrets is a map of secrets that can be used by the Steps.
	Secrets map[string]map[string]string
	// Actor is the name of the actor triggering the Promotion.
	Actor string
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

// StepEnvOption is a functional option for customizing the environment of a
// Step built by BuildEnv.
type StepEnvOption func(map[string]any)

// StepEnvWithVars returns a StepEnvOption that adds the provided vars to the
// environment of the Step.
func StepEnvWithVars(vars map[string]any) StepEnvOption {
	return func(env map[string]any) {
		env["vars"] = vars
	}
}

// StepEnvWithSecrets returns a StepEnvOption that adds the provided secrets to
// the environment of the Step.
func StepEnvWithSecrets(secrets map[string]map[string]string) StepEnvOption {
	return func(env map[string]any) {
		env["secrets"] = secrets
	}
}

// StepEnvWithStepMetas returns a StepEnvOption that adds StepExecutionMetadata
// indexed by alias to the environment of the Step.
func StepEnvWithStepMetas(promoCtx Context) StepEnvOption {
	metas := make(map[string]any, len(promoCtx.StepExecutionMetadata))
	for _, stepMeta := range promoCtx.StepExecutionMetadata {
		metas[stepMeta.Alias] = stepMeta
	}
	return func(env map[string]any) {
		env["stepMetas"] = metas
	}
}

// StepEnvWithOutputs returns a StepEnvOption that adds the provided outputs to
// the environment of the Step.
func StepEnvWithOutputs(outputs promotion.State) StepEnvOption {
	return func(env map[string]any) {
		env["outputs"] = outputs
	}
}

// StepEnvWithTaskOutputs returns a StepEnvOption that adds the provided
// task outputs to the environment of the Step.
func StepEnvWithTaskOutputs(alias string, outputs promotion.State) StepEnvOption {
	return func(env map[string]any) {
		// Ensure that if the Step originated from a task, the task outputs are
		// available to the Step. This allows inflated Steps to access the outputs
		// of the other Steps in the task without needing to know the alias
		// (namespace) of the task.
		if taskOutput := getTaskOutputs(alias, outputs); taskOutput != nil {
			env["task"] = map[string]any{
				"outputs": taskOutput,
			}
		}
	}
}

// GetTimeout returns the maximum interval the provided StepRunner may spend
// attempting to execute the Step before retries are abandoned and the entire
// Promotion is marked as failed. If the StepRunner is a RetryableStepRunner,
// its timeout is used as the default. Otherwise, the default is 0 (no limit).
func (s *Step) GetTimeout(runner promotion.StepRunner) *time.Duration {
	fallback := ptr.To(time.Duration(0))
	if retryCfg, isRetryable := runner.(promotion.RetryableStepRunner); isRetryable {
		fallback = retryCfg.DefaultTimeout()
	}
	return s.Retry.GetTimeout(fallback)
}

// GetErrorThreshold returns the number of consecutive times the provided
// StepRunner must fail to execute the Step (for any reason) before retries are
// abandoned and the entire Promotion is marked as failed. If the StepRunner is
// a RetryableStepRunner, its threshold is used as the default. Otherwise, the
// default is 1.
func (s *Step) GetErrorThreshold(runner promotion.StepRunner) uint32 {
	fallback := uint32(1)
	if retryCfg, isRetryable := runner.(promotion.RetryableStepRunner); isRetryable {
		fallback = retryCfg.DefaultErrorThreshold()
	}
	return s.Retry.GetErrorThreshold(fallback)
}

// BuildEnv returns the environment for the Step. The environment includes the
// context of the Promotion and any additional options provided (e.g. outputs,
// task outputs, vars, secrets).
//
// The environment is a (nested) map of string keys to any values. The keys are
// used as variables in the Step configuration.
func (s *Step) BuildEnv(promoCtx Context, opts ...StepEnvOption) map[string]any {
	env := map[string]any{
		"ctx": map[string]any{
			"project":   promoCtx.Project,
			"promotion": promoCtx.Promotion,
			"stage":     promoCtx.Stage,
			"targetFreight": map[string]any{
				"name": promoCtx.TargetFreightRef.Name,
				"origin": map[string]any{
					"name": promoCtx.TargetFreightRef.Origin.Name,
				},
			},
			"meta": map[string]any{
				"promotion": map[string]any{
					"actor": promoCtx.Actor,
				},
			},
		},
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(env)
	}

	return env
}

// Skip returns true if the Step should be skipped based on the If condition.
// The If condition is evaluated against the provided Context and State.
func (s *Step) Skip(
	ctx context.Context,
	cl client.Client,
	cache *gocache.Cache,
	promoCtx Context,
	state promotion.State,
) (bool, error) {
	// If no "if" condition is provided, then this step is automatically skipped
	// if any of the previous steps have errored or failed and is not skipped
	// otherwise.
	if s.If == "" {
		return promoCtx.StepExecutionMetadata.HasFailures(), nil
	}

	vars, err := s.GetVars(ctx, cl, cache, promoCtx, state)
	if err != nil {
		return false, err
	}

	env := s.BuildEnv(
		promoCtx,
		StepEnvWithStepMetas(promoCtx),
		StepEnvWithOutputs(state),
		StepEnvWithTaskOutputs(s.Alias, state),
		StepEnvWithVars(vars),
	)

	v, err := expressions.EvaluateTemplate(
		s.If,
		env,
		append(
			append(
				exprfn.FreightOperations(
					ctx,
					cl,
					promoCtx.Project,
					promoCtx.FreightRequests,
					promoCtx.Freight.References(),
				),
				exprfn.DataOperations(ctx, cl, cache, promoCtx.Project)...,
			),
			exprfn.StatusOperations(s.Alias, promoCtx.StepExecutionMetadata)...,
		)...,
	)
	if err != nil {
		return false, err
	}

	if b, ok := v.(bool); ok {
		return !b, nil
	}

	return false, fmt.Errorf("expression must evaluate to a boolean")
}

// GetConfig returns the Config unmarshalled into a map. Any expr-lang
// expressions are evaluated against the provided Context and State prior to
// unmarshaling.
func (s *Step) GetConfig(
	ctx context.Context,
	cl client.Client,
	cache *gocache.Cache,
	promoCtx Context,
	state promotion.State,
) (promotion.Config, error) {
	if s.Config == nil {
		return nil, nil
	}

	vars, err := s.GetVars(ctx, cl, cache, promoCtx, state)
	if err != nil {
		return nil, err
	}

	env := s.BuildEnv(
		promoCtx,
		StepEnvWithStepMetas(promoCtx),
		StepEnvWithOutputs(state),
		StepEnvWithTaskOutputs(s.Alias, state),
		StepEnvWithVars(vars),
		StepEnvWithSecrets(promoCtx.Secrets),
	)

	evaledCfgJSON, err := expressions.EvaluateJSONTemplate(
		s.Config,
		env,
		append(
			exprfn.FreightOperations(
				ctx,
				cl,
				promoCtx.Project,
				promoCtx.FreightRequests,
				promoCtx.Freight.References(),
			),
			exprfn.DataOperations(ctx, cl, cache, promoCtx.Project)...,
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

// GetVars returns the variables defined in the Step. The variables are
// evaluated against the provided Context.
func (s *Step) GetVars(
	ctx context.Context,
	cl client.Client,
	cache *gocache.Cache,
	promoCtx Context,
	state promotion.State,
) (map[string]any, error) {
	vars := make(map[string]any)

	// Evaluate the global variables defined in the Promotion itself, these
	// variables DO NOT have access to the (task) outputs.
	for _, v := range promoCtx.Vars {
		newVar, err := expressions.EvaluateTemplate(
			v.Value,
			s.BuildEnv(promoCtx, StepEnvWithVars(vars)),
			append(
				exprfn.FreightOperations(ctx, cl, promoCtx.Project, promoCtx.FreightRequests, promoCtx.Freight.References()),
				exprfn.DataOperations(ctx, cl, cache, promoCtx.Project)...,
			)...,
		)
		if err != nil {
			return nil, fmt.Errorf("error pre-processing promotion variable %q: %w", v.Name, err)
		}
		vars[v.Name] = newVar
	}

	// Evaluate the variables defined in the Step. These variables DO have access
	// to the (task) outputs.
	for _, v := range s.Vars {
		newVar, err := expressions.EvaluateTemplate(
			v.Value,
			s.BuildEnv(
				promoCtx,
				StepEnvWithStepMetas(promoCtx),
				StepEnvWithOutputs(state),
				StepEnvWithTaskOutputs(s.Alias, state),
				StepEnvWithVars(vars),
			),
			append(
				exprfn.FreightOperations(ctx, cl, promoCtx.Project, promoCtx.FreightRequests, promoCtx.Freight.References()),
				exprfn.DataOperations(ctx, cl, cache, promoCtx.Project)...,
			)...,
		)
		if err != nil {
			return nil, fmt.Errorf("error pre-processing promotion variable %q: %w", v.Name, err)
		}
		vars[v.Name] = newVar
	}

	return vars, nil
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
	State promotion.State
}

// getTaskOutputs returns the outputs of a task that are relevant to the current
// Step. This is useful when a Step is inflated from a task and needs to access
// the outputs of that task.
func getTaskOutputs(alias string, state promotion.State) promotion.State {
	if namespace := getAliasNamespace(alias); namespace != "" {
		taskOutputs := make(promotion.State)
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
