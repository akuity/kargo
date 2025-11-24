package external

import (
	"context"
	"fmt"
	"maps"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

type actionResult struct {
	ActionName      kargoapi.GenericWebhookActionName `json:"actionName"`
	ConditionResult conditionResult                   `json:"conditionResult"`
	TargetResults   []targetResult                    `json:"targetResults,omitempty"`
}

func newActionResult(action kargoapi.GenericWebhookAction) actionResult {
	return actionResult{
		ActionName: action.Name,
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

func handleAction(
	ctx context.Context,
	c client.Client,
	project string,
	actionEnv map[string]any,
	action kargoapi.GenericWebhookAction,
) []targetResult {
	var targetResults []targetResult
	switch action.Name {
	case kargoapi.GenericWebhookActionNameRefresh:
		targetResults = handleRefreshAction(ctx, c, project, actionEnv, action)
	}
	return targetResults
}

func handleRefreshAction(
	ctx context.Context,
	c client.Client,
	project string,
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
		objects, err := listTargetObjects(tCtx, c, project, target, actionEnv)
		if err != nil {
			tLogger.Error(err, "failed to list objects for target")
			targetResults[i].ListError = fmt.Errorf("failed to list target objects: %w", err)
			continue
		}
		targetResults[i].RefreshResults = refreshObjects(tCtx, c, target.Name, objects)
	}
	return targetResults
}
