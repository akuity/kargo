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
						"action", action.Name,
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

func newActionEnv(action kargoapi.GenericWebhookAction, globalEnv map[string]any) map[string]any {
	actionEnv := maps.Clone(globalEnv)
	for paramKey, paramValue := range action.Parameters {
		actionEnv[paramKey] = paramValue
	}
	return actionEnv
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
