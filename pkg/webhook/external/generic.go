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
	xhttp "github.com/akuity/kargo/pkg/http"
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
		env, err := newBaseEnv(requestBody, r)
		if err != nil {
			xhttp.WriteErrorJSON(w, xhttp.Error(err, http.StatusBadRequest))
			return
		}
		actionResults := make([]actionResult, len(g.config.Actions))
		for i, action := range g.config.Actions {
			actionResults[i] = g.handleAction(ctx, action, env)
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
			"params":  r.URL.Query().Get,
		},
	}, nil
}

func shouldReportAsError(actionResults []actionResult) bool {
	return slices.ContainsFunc(actionResults, func(ar actionResult) bool {
		return hasErrors(ar)
	})
}

func hasErrors(ar actionResult) bool {
	if ar.ConditionResult.EvalError != "" {
		return true
	}
	if ar.ListResult != nil && len(ar.ListResult.Errors) > 0 {
		return true
	}
	refreshFailure := func(rr refreshResult) bool { return rr.Failure != "" }
	return slices.ContainsFunc(ar.RefreshResults, refreshFailure)
}
