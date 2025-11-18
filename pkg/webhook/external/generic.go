package external

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"

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
	defaultWebhookReceiverRegistry.MustRegister(
		webhookReceiverRegistration{
			Predicate: func(_ context.Context, cfg kargoapi.WebhookReceiverConfig) (bool, error) {
				return cfg.Generic != nil, nil
			},
			Value: newGenericWebhookReceiver,
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

		baseEnv, err := newBaseEnv(requestBody, r)
		if err != nil {
			logger.Error(err, "error creating base environment")
			// this can only fail if the request body is invalid json
			xhttp.WriteErrorJSON(w, xhttp.Error(err, http.StatusBadRequest))
			return
		}

		actionResults := make([]actionResult, len(g.config.Actions))
		for i, action := range g.config.Actions {
			aLogger := logger.WithValues(
				"action", action.Name,
				"expression", action.MatchExpression,
			)

			actionResults[i] = newActionResult(action)
			satisfied, err := conditionSatisfied(action.MatchExpression, baseEnv)
			if err != nil {
				aLogger.Error(err, "failed to evaluate criteria; skipping action")
				actionResults[i].ConditionResult.EvalError = err.Error()
				continue
			}

			actionResults[i].ConditionResult.Satisfied = satisfied
			if !satisfied {
				aLogger.Info("condition not satisfied; skipping action")
				continue
			}

			actionResults[i].TargetResults = handleAction(
				logging.ContextWithLogger(ctx, aLogger),
				g.client,
				g.project,
				newActionEnv(action.Parameters, baseEnv),
				action,
			)
		}
		statusCode := http.StatusOK
		if shouldReportAsError(actionResults) {
			statusCode = http.StatusInternalServerError
		}
		xhttp.WriteResponseJSON(w, statusCode,
			map[string]any{"actionResults": actionResults},
		)
	})
}

func newBaseEnv(requestBody []byte, r *http.Request) (map[string]any, error) {
	var body any
	if err := json.Unmarshal(requestBody, &body); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}
	return map[string]any{
		"normalizeGit":   urls.NormalizeGit,
		"normalizeImage": urls.NormalizeImage,
		"normalizeChart": urls.NormalizeChart,
		"request": map[string]any{
			"header":  r.Header.Get,
			"headers": r.Header.Values,
			"body":    body,
			"method":  r.Method,
			"url":     r.URL.String(),
		},
	}, nil
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

// should reportAsError determines whether the overall webhook processing should be
// reported as an error based on the action results. Returns true if any action
// resulted in an expression evaluation error or any of its underlying
// targets had errors.
func shouldReportAsError(actionResults []actionResult) bool {
	return slices.ContainsFunc(actionResults, func(ar actionResult) bool {
		return ar.ConditionResult.EvalError != "" || hasTargetErrors(ar)
	})
}

func hasTargetErrors(ar actionResult) bool {
	for _, tr := range ar.TargetResults {
		if tr.ListError != nil {
			return true
		}
		switch ar.ActionName {
		case kargoapi.GenericWebhookActionNameRefresh:
			refreshFailure := func(rr refreshResult) bool { return rr.Failure != "" }
			if slices.ContainsFunc(tr.RefreshResults, refreshFailure) {
				return true
			}
		}
	}
	return false
}
