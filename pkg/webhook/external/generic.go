package external

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/expressions"
	xhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/indexer"
	"github.com/akuity/kargo/pkg/logging"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/selection"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

var supportedIndices = []string{
	indexer.WarehousesBySubscribedURLsField,
}

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

		var body any
		if err := json.Unmarshal(requestBody, &body); err != nil {
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					fmt.Errorf("invalid request body: %w", err),
					http.StatusBadRequest,
				),
			)
			return
		}

		// Shared environment for all actions.
		globalEnv := map[string]any{
			"request": map[string]any{
				"header":  r.Header.Get,
				"headers": r.Header.Values,
				"body":    body,
				"method":  r.Method,
				"url":     r.URL.String(),
			},
		}

		for _, action := range g.config.Actions {
			switch action.Action {
			case kargoapi.GenericWebhookActionNameRefresh:
				actionEnv := newActionEnv(action, globalEnv)
				conditionsMet, err := conditionsMet(ctx, action.MatchConditions, actionEnv)
				if err != nil {
					logger.Error(err, "error evaluating match conditions for refresh action")
					xhttp.WriteErrorJSON(w,
						xhttp.Error(
							fmt.Errorf("error evaluating match conditions for refresh action: %w", err),
							http.StatusBadRequest,
						),
					)
					return
				}
				if !conditionsMet {
					logger.Info("match conditions not met; skipping refresh action", "action", action)
					continue
				}
				for _, target := range action.Targets {
					_, err := g.buildListOptionsForTarget(target, actionEnv)
					if err != nil {
						logger.Error(err, "failed to build list options for warehouse target")
						continue
					}
					// TODO(Faris): pass listOpts to generic refresh func
				}
			}
		}
	})
}

func (g *genericWebhookReceiver) buildListOptionsForTarget(
	t kargoapi.GenericWebhookTarget,
	env map[string]any,
) ([]client.ListOption, error) {
	listOpts := []client.ListOption{client.InNamespace(g.project)}

	if len(t.IndexSelector.MatchExpressions) > 0 {
		indexSelectorListOpts, err := g.newListOptionsForIndexSelector(t.IndexSelector, env)
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

func (g *genericWebhookReceiver) newListOptionsForIndexSelector(
	is kargoapi.IndexSelector,
	env map[string]any,
) ([]client.ListOption, error) {
	var listOpts []client.ListOption
	for _, expr := range is.MatchExpressions {
		parsed, err := parseValuesAsList(&expr.Values)
		if err != nil {
			return nil, fmt.Errorf("failed to parse values for index selector expression: %w", err)
		}

		result, err := expressions.EvaluateTemplate(parsed, env)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate values expression for index selector: %w", err)
		}

		resultAsList, ok := result.([]string)
		if !ok {
			return nil, fmt.Errorf("index selector values expression evaluated to %T; expected []string", result)
		}

		listOpts = append(listOpts, client.MatchingFields{
			// the key is the index name
			expr.Key: resultAsList,
		})
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

func conditionsMet(
	ctx context.Context,
	conditions []kargoapi.ConditionSelector,
	env map[string]any,
) (bool, error) {
	logger := logging.LoggerFromContext(ctx)
	for _, condition := range conditions {
		conditionMet, err := evaluateCondition(condition.Expression, env)
		if err != nil {
			logger.Error(err, "failed to evaluate condition", "condition", condition.Name)
			continue
		}
		if !conditionMet {
			logger.Debug("condition not met", "condition", condition.Name)
			return false, nil
		}
	}
	return true, nil
}

func evaluateCondition(expression string, env map[string]any) (bool, error) {
	result, err := expressions.EvaluateTemplate(expression, env)
	if err != nil {
		return false, fmt.Errorf("failed to compile expression: %w", err)
	}
	met, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("expression result %q evaluated to %T; not a boolean", result, result)
	}
	return met, nil
}

func parseValuesAsList(values *apiextensionsv1.JSON) ([]string, error) {
	if values == nil {
		return nil, nil
	}

	// Try to unmarshal as array first. If this succeeds, we are dealing with a list.
	var list []string
	if err := yaml.Unmarshal(values.Raw, &list); err == nil {
		return list, nil
	}

	// Not a list. Assume we are dealing with an expr-lang string that we
	// have to "unpack" into a list.
	var expr string
	if err := yaml.Unmarshal(values.Raw, &expr); err != nil {
		return nil, fmt.Errorf("values must be either a string or array of strings: %w", err)
	}

	// TODO: now that we have an expr-lang string, evaluate it using the
	// library combined with request data to get a YAML or JSON "list".

	// TODO: after evaluating the expression, attempt to unmarshal this
	// again into a []string.
	var exprList []string
	return exprList
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
