package external

import (
	"context"
	"fmt"
	"maps"

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
	ListError      error                             `json:"listError,omitempty"`
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
	env map[string]any,
	action kargoapi.GenericWebhookAction,
) []targetResult {
	var targetResults []targetResult
	switch action.ActionType {
	case kargoapi.GenericWebhookActionTypeRefresh:
		targetResults = g.handleRefreshAction(ctx, env, action)
	}
	return targetResults
}

func (g *genericWebhookReceiver) handleRefreshAction(
	ctx context.Context,
	actionEnv map[string]any,
	action kargoapi.GenericWebhookAction,
) []targetResult {
	logger := logging.LoggerFromContext(ctx)
	targetResults := make([]targetResult, len(action.Targets))
	for i, target := range action.Targets {
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
			targetResults[i].ListError = fmt.Errorf("failed to list target objects: %w", err)
			continue
		}
		targetResults[i].RefreshResults = refreshObjects(tCtx, g.client, objects)
	}
	return targetResults
}
