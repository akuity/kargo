package external

import (
	"context"
	"fmt"
	"maps"

	"github.com/expr-lang/expr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/logging"
)

const (
	resultNotApplicable  = "NotApplicable"
	resultError          = "Error"
	resultSuccess        = "Success"
	resultPartialSuccess = "PartialSuccess"
	resultFailure        = "Failure"
)

const (
	summaryRequestNotMatched      = "Request did not match whenExpression"
	summaryRequestMatchingError   = "Error evaluating whenExpression"
	summaryResourceSelectionError = "Error evaluating targetSelectionCriteria"
)

type actionResult struct {
	kargoapi.GenericWebhookAction `json:",inline"`
	MatchedWhenExpression         bool             `json:"matchedWhenExpression"`
	SelectedTargets               []selectedTarget `json:"selectedTargets,omitempty"`
	Result                        string           `json:"result,omitempty"`
	Summary                       string           `json:"summary,omitempty"`
}

type selectedTarget struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Success   bool   `json:"success"`
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
	ar := actionResult{GenericWebhookAction: action}
	env = newActionEnv(action.Parameters, env)
	aLogger := logging.LoggerFromContext(ctx).WithValues(
		"action", action.ActionType,
		"whenExpression", action.WhenExpression,
	)
	met, err := whenExpressionMet(action.WhenExpression, env)
	if err != nil {
		aLogger.Error(err, "failed to evaluate whenExpression; skipping action")
		ar.MatchedWhenExpression = false
		ar.Result = resultError
		ar.Summary = summaryRequestMatchingError
		return ar
	}
	ar.MatchedWhenExpression = met
	if !met {
		aLogger.Debug("whenExpression not satisfied; skipping action")
		ar.Result = resultNotApplicable
		ar.Summary = summaryRequestNotMatched
		return ar
	}
	objects, err := g.listUniqueObjects(ctx, action, env)
	if err != nil {
		aLogger.Error(err, "failed to list unique objects")
		ar.Result = resultError
		ar.Summary = summaryResourceSelectionError
		return ar
	}
	switch action.ActionType {
	case kargoapi.GenericWebhookActionTypeRefresh:
		ar.SelectedTargets, ar.Result, ar.Summary = refreshObjects(ctx, g.client, objects)
	}
	return ar
}

func whenExpressionMet(expression string, env map[string]any) (bool, error) {
	if expression == "" {
		return true, nil
	}
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
