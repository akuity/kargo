package external

import (
	"context"
	"fmt"
	"maps"

	"github.com/expr-lang/expr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

type actionResult struct {
	ActionType      kargoapi.GenericWebhookActionType `json:"actionType"`
	ConditionResult conditionResult                   `json:"conditionResult"`
	TargetResults   []targetResult                    `json:"targetResults,omitempty"`
}

func newActionResult(action kargoapi.GenericWebhookAction) actionResult {
	return actionResult{
		ActionType: action.ActionType,
		ConditionResult: conditionResult{
			Expression: action.MatchExpression,
		},
	}
}

type conditionResult struct {
	Expression string `json:"expression"`
	Satisfied  bool   `json:"satisfied"`
	EvalError  string `json:"evalError,omitempty"`
}

type targetResult struct {
	Kind           kargoapi.GenericWebhookTargetKind `json:"kind"`
	Name           string                            `json:"name,omitempty"`
	ListError      string                            `json:"listError,omitempty"`
	RefreshResults []refreshResult                   `json:"refreshResults,omitempty"`
}

func newActionEnv(params map[string]string, baseEnv map[string]any) map[string]any {
	actionEnv := maps.Clone(baseEnv)
	m := make(map[string]any, len(params))
	for paramKey, paramValue := range params {
		m[paramKey] = paramValue
	}
	actionEnv["params"] = m
	return actionEnv
}

func (g *genericWebhookReceiver) handleAction(
	ctx context.Context,
	action kargoapi.GenericWebhookAction,
	env map[string]any,
) actionResult {
	ar := newActionResult(action)
	env = newActionEnv(action.Parameters, env)
	aLogger := logging.LoggerFromContext(ctx).WithValues(
		"action", action.ActionType,
		"expression", action.MatchExpression,
	)

	satisfied, err := conditionSatisfied(action.MatchExpression, env)
	if err != nil {
		aLogger.Error(err, "failed to evaluate criteria; skipping action")
		ar.ConditionResult.EvalError = err.Error()
		return ar
	}

	ar.ConditionResult.Satisfied = satisfied
	if !satisfied {
		aLogger.Debug("condition not satisfied; skipping action")
		return ar
	}

	switch action.ActionType {
	case kargoapi.GenericWebhookActionTypeRefresh:
		ar.TargetResults = g.handleRefreshAction(ctx, env, action)
	}
	return ar
}

func (g *genericWebhookReceiver) handleRefreshAction(
	ctx context.Context,
	actionEnv map[string]any,
	action kargoapi.GenericWebhookAction,
) []targetResult {
	logger := logging.LoggerFromContext(ctx)
	targetResults := make([]targetResult, len(action.TargetSelectionCriteria))
	for i, target := range action.TargetSelectionCriteria {
		tLogger := logger.WithValues(
			"targetKind", target.Kind,
			"targetName", target.Name,
		)
		targetResults[i] = targetResult{
			Kind: target.Kind,
			Name: target.Name,
		}
		tCtx := logging.ContextWithLogger(ctx, tLogger)
		objects, err := g.listTargetObjects(tCtx, target, actionEnv)
		if err != nil {
			tLogger.Error(err, "failed to list objects for target")
			targetResults[i].ListError = fmt.Sprintf("failed to list target objects: %v", err)
			continue
		}
		targetResults[i].RefreshResults = refreshObjects(tCtx, g.client, objects)
	}
	logger.Info("checking target results", "results", targetResults)
	return targetResults
}

func conditionSatisfied(expression string, env map[string]any) (bool, error) {
	program, err := expr.Compile(expression)
	if err != nil {
		return false, err
	}
	result, err := expr.Run(program, env)
	if err != nil {
		return false, err
	}
	satisfied, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("match expression result %q is of type %T; expected bool", result, result)
	}
	return satisfied, nil
}
