package external

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/expressions"
	xhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/urls"
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
			// this can only fail if the request body is invalid json
			xhttp.WriteErrorJSON(w, xhttp.Error(err, http.StatusBadRequest))
			return
		}

		results := make([]actionResult, len(g.config.Actions))
		for i, action := range g.config.Actions {
			results[i].ActionName = action.Name
			// append action specific parameters to a copy of the global env
			actionEnv := newActionEnv(action, globalEnv)
			if satisfied, err := conditionSatisfied(action.MatchExpression, actionEnv); err != nil || !satisfied {
				logger.Info("match expression not met; skipping refresh action",
					"action", action.Name,
					"expression", action.MatchExpression,
					"satisfied", satisfied,
					"evalError", err,
				)
				results[i].Condition = &conditionResult{
					Expression: action.MatchExpression,
					Satisfied:  satisfied,
					EvalError:  fmt.Sprintf("%v", err),
				}
				continue
			}
			switch action.Name {
			case kargoapi.GenericWebhookActionNameRefresh:
				results[i].RefreshResults = handleRefreshAction(
					ctx, g.client, g.project, actionEnv, action.Targets,
				)
			}
			// add new action handlers here
		}
		resp := map[string]any{"results": results}
		if shouldReportAsError(results) {
			xhttp.WriteResponseJSON(w, http.StatusInternalServerError, resp)
			return
		}
		xhttp.WriteResponseJSON(w, http.StatusOK, resp)
	})
}

type actionResult struct {
	ActionName     kargoapi.GenericWebhookActionName `json:"actionName"`
	Condition      *conditionResult                  `json:"conditionFailure,omitempty"`
	RefreshResults []refreshTargetResult             `json:"refreshResults,omitempty"`
}

type conditionResult struct {
	Expression string `json:"expression"`
	Satisfied  bool   `json:"satisfied"`
	EvalError  string `json:"evalError"`
}

func newGlobalEnv(requestBody []byte, r *http.Request) (map[string]any, error) {
	var body any
	if err := json.Unmarshal(requestBody, &body); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}
	return map[string]any{
		"normalize": func(urlType, url string) string {
			switch urlType {
			case "git":
				return urls.NormalizeGit(url)
			case "image":
				return urls.NormalizeImage(url)
			case "chart":
				return urls.NormalizeChart(url)
			default:
				return url
			}
		},
		"request": map[string]any{
			"header":  r.Header.Get,
			"headers": r.Header.Values,
			"body":    body,
			"method":  r.Method,
			"url":     r.URL.String(),
		},
	}, nil
}

func newActionEnv(action kargoapi.GenericWebhookAction, globalEnv map[string]any) map[string]any {
	actionEnv := maps.Clone(globalEnv)
	m := make(map[string]any, len(action.Parameters))
	for paramKey, paramValue := range action.Parameters {
		m[paramKey] = paramValue
	}
	actionEnv["params"] = m
	return actionEnv
}

func conditionSatisfied(expression string, env map[string]any) (bool, error) {
	result, err := expressions.EvaluateTemplate(expression, env)
	if err != nil {
		return false, err
	}
	satisfied, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("match expression result %q is of type %T; expected bool", result, result)
	}
	return satisfied, nil
}

func shouldReportAsError(results []actionResult) bool {
	for _, result := range results {
		// eval errors should be treated as 500s
		if result.Condition != nil && result.Condition.EvalError != "<nil>" {
			return true
		}
		for _, refreshResult := range result.RefreshResults {
			// building list options and failing to list
			if refreshResult.Err != nil {
				return true
			}
			for _, whr := range refreshResult.WarehouseRefreshResults {
				// refresh failures
				if whr.Failure != "" {
					return true
				}
			}
		}
	}
	return false
}
