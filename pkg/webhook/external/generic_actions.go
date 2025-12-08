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
	ListResult      *listResult                       `json:"listResult,omitempty"`
	RefreshResults  []refreshResult                   `json:"refreshResults,omitempty"`
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
	objects, errs := g.listUniqueObjects(ctx, action, env)
	if len(errs) > 0 {
		aLogger.Debug("list errors detected",
			"numErrors", len(errs),
			"errors", errs,
		)
	}
	ar.ListResult = newListResult(objects, errs)
	println("in handleAction, listResult has", len(ar.ListResult.Errors), "errors")
	switch action.ActionType {
	case kargoapi.GenericWebhookActionTypeRefresh:
		ar.RefreshResults = refreshObjects(ctx, g.client, objects)
	}
	return ar
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
