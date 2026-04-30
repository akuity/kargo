package promotion

import (
	"context"
	"fmt"
	"slices"
	"strings"

	gocache "github.com/patrickmn/go-cache"
	"go.yaml.in/yaml/v3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/expressions"
	exprfn "github.com/akuity/kargo/pkg/expressions/function"
	"github.com/akuity/kargo/pkg/kargo"
)

// StepEvaluator handles the evaluation and processing of Promotion steps,
// including conditional logic, configuration templating, and context building.
// It centralizes all complex expression evaluation logic that determines how
// steps should be executed based on the current state of the Promotion.
//
// The processor maintains a shared cache to improve performance when multiple
// expressions within the same step access the same data, such as Secrets or
// ConfigMaps.
type StepEvaluator struct {
	client client.Client
	cache  *gocache.Cache
}

// NewStepEvaluator creates a new StepEvaluator instance with the provided
// Kubernetes client and cache. The cache is optional, and can be used to
// store Kubernetes objects that are frequently accessed by the expression
// evaluation logic, such as Secrets and ConfigMaps, to avoid unnecessary API
// calls and improve performance.
func NewStepEvaluator(cl client.Client, cache *gocache.Cache) *StepEvaluator {
	return &StepEvaluator{
		client: cl,
		cache:  cache,
	}
}

// ExprEnvOption is a functional option for customizing the expression language
// environment built by the StepEvaluator for a Step.
type ExprEnvOption func(map[string]any)

// ExprEnvWithVars returns a ExprEnvOption that adds the provided vars to the
// expression language environment of the Step.
func ExprEnvWithVars(vars map[string]any) ExprEnvOption {
	return func(env map[string]any) {
		env["vars"] = vars
	}
}

// ExprEnvWithStepMetas returns a ExprEnvOption that adds StepExecutionMetadata
// indexed by alias to the expression language environment of the Step.
func ExprEnvWithStepMetas(promoCtx Context) ExprEnvOption {
	metas := make(map[string]any, len(promoCtx.StepExecutionMetadata))
	for _, stepMeta := range promoCtx.StepExecutionMetadata {
		metas[stepMeta.Alias] = stepMeta
	}
	return func(env map[string]any) {
		env["stepMetas"] = metas
	}
}

// ExprEnvWithOutputs returns a ExprEnvOption that adds the provided outputs to
// the expression language environment of the Step.
func ExprEnvWithOutputs(outputs State) ExprEnvOption {
	return func(env map[string]any) {
		env["outputs"] = outputs
	}
}

// ExprEnvWithTaskOutputs returns a ExprEnvOption that adds the provided
// task outputs to the expression language environment of the Step.
func ExprEnvWithTaskOutputs(alias string, outputs State) ExprEnvOption {
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

// BuildExprEnv builds an environment map for evaluating expressions in
// Promotion steps. The environment includes context information, such as the
// Project, Promotion, Stage, and target Freight reference.
//
// The environment can be extended with additional options provided via the
// ExprEnvOption functional options. These options can be used to add variables
// or modify the expression language environment.
func (p *StepEvaluator) BuildExprEnv(promoCtx Context, opts ...ExprEnvOption) map[string]any {
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

// Vars evaluates the variables defined in the Context and Step, returning
// a map of variable names to their evaluated values. The variables defined in
// the Context are evaluated first, followed by the variables defined in the
// Step.
//
// The variables defined in the Context do not have access to the outputs of
// the Step, while the variables defined in the Step do have access to the
// outputs of any preceding Steps.
func (p *StepEvaluator) Vars(ctx context.Context, promoCtx Context, step Step) (map[string]any, error) {
	vars := make(map[string]any)

	// Prepare the expression functions that will be available during variable
	// evaluation. These functions provide access to data operations, freight
	// operations, and utility functions.
	exprOpts := slices.Concat(
		exprfn.DataOperations(ctx, p.client, p.cache, promoCtx.Project),
		exprfn.FreightOperations(
			ctx, p.client, promoCtx.Project, promoCtx.FreightRequests, promoCtx.Freight.References(),
		),
		exprfn.UtilityOperations(),
	)

	// Evaluate the global variables defined in the Promotion itself, these
	// variables DO NOT have access to the (task) outputs.
	for _, v := range promoCtx.Vars {
		newVar, err := expressions.EvaluateTemplate(
			v.Value,
			p.BuildExprEnv(promoCtx, ExprEnvWithVars(vars)),
			exprOpts...,
		)
		if err != nil {
			return nil, fmt.Errorf("error pre-processing promotion variable %q: %w", v.Name, err)
		}
		vars[v.Name] = newVar
	}

	// Evaluate the variables defined in the Step. These variables DO have access
	// to the (task) outputs.
	for _, v := range step.Vars {
		newVar, err := expressions.EvaluateTemplate(
			v.Value,
			p.BuildExprEnv(
				promoCtx,
				ExprEnvWithStepMetas(promoCtx),
				ExprEnvWithOutputs(promoCtx.State),
				ExprEnvWithTaskOutputs(step.Alias, promoCtx.State),
				ExprEnvWithVars(vars),
			),
			exprOpts...,
		)
		if err != nil {
			return nil, fmt.Errorf("error pre-processing promotion variable %q: %w", v.Name, err)
		}
		vars[v.Name] = newVar
	}

	return vars, nil
}

// ShouldSkip determines whether a Step should be skipped based on the "if"
// condition defined in the Step. If the "if" condition evaluates to false, the
// Step is skipped. If the "if" condition is not defined, the Step is skipped
// if any of the previous Steps have failed or errored and are not skipped
// otherwise.
func (p *StepEvaluator) ShouldSkip(ctx context.Context, promoCtx Context, step Step) (bool, error) {
	// If no "if" condition is provided, then this step is automatically skipped
	// if any of the previous steps have errored or failed and is not skipped
	// otherwise.
	if step.If == "" {
		return promoCtx.StepExecutionMetadata.HasFailures(), nil
	}

	vars, err := p.Vars(ctx, promoCtx, step)
	if err != nil {
		return false, err
	}

	env := p.BuildExprEnv(
		promoCtx,
		ExprEnvWithStepMetas(promoCtx),
		ExprEnvWithOutputs(promoCtx.State),
		ExprEnvWithTaskOutputs(step.Alias, promoCtx.State),
		ExprEnvWithVars(vars),
	)

	v, err := expressions.EvaluateTemplate(
		step.If,
		env,
		slices.Concat(
			exprfn.DataOperations(ctx, p.client, p.cache, promoCtx.Project),
			exprfn.FreightOperations(
				ctx,
				p.client,
				promoCtx.Project,
				promoCtx.FreightRequests,
				promoCtx.Freight.References(),
			),
			exprfn.StatusOperations(step.Alias, promoCtx.StepExecutionMetadata),
			exprfn.UtilityOperations(),
		)...,
	)
	if err != nil {
		return false, err
	}

	switch val := v.(type) {
	case bool:
		return !val, nil
	default:
		// If the expression does not evaluate to a boolean, we return an error.
		return false, fmt.Errorf("expression must evaluate to a boolean, got %T", v)
	}
}

// Config evaluates the configuration defined in the Step, returning a map of
// configuration keys to their evaluated values. The configuration is evaluated
// in the context of the Context and Step, allowing for dynamic configuration
// based on the current state of the Promotion.
func (p *StepEvaluator) Config(ctx context.Context, promoCtx Context, step Step) (Config, error) {
	if step.Config == nil {
		return nil, nil
	}

	vars, err := p.Vars(ctx, promoCtx, step)
	if err != nil {
		return nil, err
	}

	env := p.BuildExprEnv(
		promoCtx,
		ExprEnvWithStepMetas(promoCtx),
		ExprEnvWithOutputs(promoCtx.State),
		ExprEnvWithTaskOutputs(step.Alias, promoCtx.State),
		ExprEnvWithVars(vars),
	)

	evaledCfgJSON, err := expressions.EvaluateJSONTemplate(
		step.Config,
		env,
		slices.Concat(
			exprfn.FreightOperations(
				ctx,
				p.client,
				promoCtx.Project,
				promoCtx.FreightRequests,
				promoCtx.Freight.References(),
			),
			exprfn.DataOperations(ctx, p.client, p.cache, promoCtx.Project),
			exprfn.StatusOperations(step.Alias, promoCtx.StepExecutionMetadata),
			exprfn.UtilityOperations(),
		)...,
	)
	if err != nil {
		return nil, err
	}

	var config map[string]any
	if err = yaml.Unmarshal(evaledCfgJSON, &config); err != nil {
		return nil, nil
	}
	return config, nil
}

// BuildStepContext builds a StepContext for the given Step in the context of
// the provided Promotion context. The StepContext contains all the necessary
// information needed to execute the Step, including the evaluated configuration,
// the shared state, and the alias of the Step.
func (p *StepEvaluator) BuildStepContext(
	ctx context.Context,
	promoCtx Context,
	step Step,
) (*StepContext, error) {
	stepCfg, err := p.Config(ctx, promoCtx, step)
	if err != nil {
		return nil, fmt.Errorf("failed to get step config: %w", err)
	}

	var freightRequests []kargoapi.FreightRequest
	if promoCtx.FreightRequests != nil {
		freightRequests = make([]kargoapi.FreightRequest, len(promoCtx.FreightRequests))
		for i, fr := range promoCtx.FreightRequests {
			freightRequests[i] = *fr.DeepCopy()
		}
	}

	return &StepContext{
		UIBaseURL:        promoCtx.UIBaseURL,
		WorkDir:          promoCtx.WorkDir,
		SharedState:      promoCtx.State.DeepCopy(),
		Alias:            step.Alias,
		Config:           stepCfg,
		Project:          promoCtx.Project,
		Stage:            promoCtx.Stage,
		Promotion:        promoCtx.Promotion,
		PromotionActor:   promoCtx.Actor,
		FreightRequests:  freightRequests,
		Freight:          *promoCtx.Freight.DeepCopy(),
		TargetFreightRef: promoCtx.TargetFreightRef,
	}, nil
}

// getTaskOutputs returns the outputs of a task that are relevant to the current
// Step. This is useful when a Step is inflated from a task and needs to access
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
