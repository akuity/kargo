package external

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/expressions"
	xhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/urls"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/selection"
)

const (
	genericSecretDataKey = "secret"
	generic              = "generic"
)

func init() {
	registry.register(
		generic,
		webhookReceiverRegistration{
			predicate: func(cfg kargoapi.WebhookReceiverConfig) bool {
				return cfg.Generic != nil
			},
			factory: newGenericWebhookReceiver,
		},
	)
}

// genericWebhookReceiver is an implementation of WebhookReceiver that
// handles inbound webhook events from generic providers.
type genericWebhookReceiver struct {
	*baseWebhookReceiver
	config *kargoapi.GenericWebhookReceiverConfig
}

// newGenericWebhookReceiver returns a new instance of genericWebhookReceiver.
func newGenericWebhookReceiver(
	c client.Client,
	project string,
	cfg kargoapi.WebhookReceiverConfig,
) WebhookReceiver {
	return &genericWebhookReceiver{
		baseWebhookReceiver: &baseWebhookReceiver{
			client:     c,
			project:    project,
			secretName: cfg.Generic.SecretRef.Name,
		},
		config: cfg.Generic,
	}
}

// getReceiverType implements WebhookReceiver.
func (g *genericWebhookReceiver) getReceiverType() string {
	return generic
}

// getSecretValues implements WebhookReceiver.
func (g *genericWebhookReceiver) getSecretValues(
	secretData map[string][]byte,
) ([]string, error) {
	secretValue, ok := secretData[genericSecretDataKey]
	if !ok {
		return nil,
			errors.New("secret data is not valid for a Generic WebhookReceiver")
	}
	return []string{string(secretValue)}, nil
}

// getHandler implements WebhookReceiver.
func (g *genericWebhookReceiver) getHandler(requestBody []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.LoggerFromContext(ctx)
		ctx = logging.ContextWithLogger(ctx, logger)

		// Shared environment for all actions.
		globalEnv, err := newGlobalEnv(requestBody, r)
		if err != nil {
			logger.Error(err, "error creating global environment")
			xhttp.WriteErrorJSON(w, err)
			return
		}

		results := make([]actionResult, len(g.config.Actions))
		for i, action := range g.config.Actions {
			results[i].ActionName = action.Name
			switch action.Name {
			case kargoapi.GenericWebhookActionNameRefresh:
				// append action specific parameters to a copy of the global env
				actionEnv := newActionEnv(action, globalEnv)
				if met, err := conditionMet(action.MatchExpression, actionEnv); err != nil || !met {
					logger.Info("match expression not met; skipping refresh action",
						"action", action,
						"expression", action.MatchExpression,
					)
					results[i].ConditionFailure = conditionResult{
						Expression: action.MatchExpression,
						Met:        met,
						Error:      err,
					}
					continue
				}
				results[i].RefreshResults = handleRefreshAction(
					ctx, g.client, g.project, actionEnv, action.Targets,
				)
			}
			// add new action handlers here
		}
		xhttp.WriteResponseJSON(w, http.StatusOK, map[string]any{"results": results})
	})
}

type actionResult struct {
	ActionName       kargoapi.GenericWebhookActionName `json:"actionName"`
	ConditionFailure conditionResult                   `json:"conditionFailure,omitempty"`
	RefreshResults   []refreshTargetResult             `json:"refreshResults,omitempty"`
}

type conditionResult struct {
	Expression string `json:"expression"`
	Met        bool   `json:"met"`
	Error      error  `json:"error,omitempty"`
}

func conditionMet(expression string, env map[string]any) (bool, error) {
	result, err := expressions.EvaluateTemplate(expression, env)
	if err != nil {
		return false, fmt.Errorf("failed to evaluate match expression: %w", err)
	}
	met, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("match expression did not evaluate to a boolean")
	}
	return met, nil
}

func newGlobalEnv(requestBody []byte, r *http.Request) (map[string]any, error) {
	var body any
	if err := json.Unmarshal(requestBody, &body); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}
	return map[string]any{
		"normalize": urls.Normalize,
		"request": map[string]any{
			"header":  r.Header.Get,
			"headers": r.Header.Values,
			"body":    body,
			"method":  r.Method,
			"url":     r.URL.String(),
		},
	}, nil
}

func buildListOptionsForTarget(
	project string,
	t kargoapi.GenericWebhookTarget,
	env map[string]any,
) ([]client.ListOption, error) {
	listOpts := []client.ListOption{client.InNamespace(project)}

	if len(t.IndexSelector.MatchExpressions) > 0 {
		indexSelectorListOpts, err := newListOptionsForIndexSelector(t.IndexSelector, env)
		if err != nil {
			return nil, fmt.Errorf("failed to create field selector: %w", err)
		}
		listOpts = append(listOpts, indexSelectorListOpts...)
	}

	if len(t.LabelSelector.MatchLabels) > 0 || len(t.LabelSelector.MatchExpressions) > 0 {
		labelSelectorListOpts, err := newListOptionsForLabelSelector(t.LabelSelector)
		if err != nil {
			return nil, fmt.Errorf("failed to create label selector: %w", err)
		}
		listOpts = append(listOpts, labelSelectorListOpts...)
	}
	return listOpts, nil
}

func newListOptionsForIndexSelector(
	is kargoapi.IndexSelector,
	env map[string]any,
) ([]client.ListOption, error) {
	var listOpts []client.ListOption
	for _, expr := range is.MatchExpressions {
		// If '${{' is not included, expressions.EvaluateTemplate will return 'value' as is.
		result, err := expressions.EvaluateTemplate(expr.Value, env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate values expression: %w", err)
		}

		resultStr, ok := result.(string)
		if !ok {
			return nil, fmt.Errorf("expression result %q evaluated to %T; not a string", result, result)
		}

		var s fields.Selector
		switch expr.Operator {
		case kargoapi.IndexSelectorRequirementOperatorEqual:
			s = fields.OneTermEqualSelector(expr.Key, resultStr)
		case kargoapi.IndexSelectorRequirementOperatorNotEqual:
			s = fields.OneTermNotEqualSelector(expr.Key, resultStr)
		default:
			return nil, fmt.Errorf("unsupported operator %q in index selector expression", expr.Operator)
		}
		listOpts = append(listOpts, client.MatchingFieldsSelector{Selector: s})
	}
	return listOpts, nil
}

func newActionEnv(action kargoapi.GenericWebhookAction, globalEnv map[string]any) map[string]any {
	actionEnv := maps.Clone(globalEnv)
	for paramKey, paramValue := range action.Parameters {
		actionEnv[paramKey] = paramValue
	}
	return actionEnv
}

func newListOptionsForLabelSelector(ls metav1.LabelSelector) ([]client.ListOption, error) {
	var requirements []labels.Requirement
	for _, e := range ls.MatchExpressions {
		op, err := labelOpToSelectionOp(e.Operator)
		if err != nil {
			return nil, fmt.Errorf("failed to convert label selector operator: %w", err)
		}
		req, err := labels.NewRequirement(e.Key, op, e.Values)
		if err != nil {
			return nil, fmt.Errorf("failed to create label requirement: %w", err)
		}
		requirements = append(requirements, *req)
	}
	for k, v := range ls.MatchLabels {
		req, err := labels.NewRequirement(k, selection.Equals, []string{v})
		if err != nil {
			return nil, fmt.Errorf("failed to create label requirement: %w", err)
		}
		requirements = append(requirements, *req)
	}
	return []client.ListOption{
		client.MatchingLabelsSelector{
			Selector: labels.NewSelector().Add(requirements...),
		},
	}, nil
}

// labelOpToSelectionOp converts a metav1.LabelSelectorOperator
// into a selection.Operator, which is used to build label requirements.
// Returns an error if the operator is not recognized. GT and LT operators
// are not supported.
func labelOpToSelectionOp(op metav1.LabelSelectorOperator) (selection.Operator, error) {
	switch op {
	case metav1.LabelSelectorOpIn:
		return selection.In, nil
	case metav1.LabelSelectorOpNotIn:
		return selection.NotIn, nil
	case metav1.LabelSelectorOpExists:
		return selection.Exists, nil
	case metav1.LabelSelectorOpDoesNotExist:
		return selection.DoesNotExist, nil
	default:
		// selection.GreaterThan & selection.LessThan don't
		// have a label selector equivalent so they're not supported.
		return "", fmt.Errorf("unsupported LabelSelectorOperator: %q", op)
	}
}
