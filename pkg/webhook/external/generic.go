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

// genericWebhookReceiver is an implementation of WebhookReceiver that can be
// configured to respond to any arbitrary POST by applying user-defined actions
// on user-defined sets of resources selected by name, labels and/or values in
// pre-built indices.
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
			map[string]any{"results": actionResults},
		)
	})
}

func newBaseEnv(requestBody []byte, r *http.Request) (map[string]any, error) {
	var body any
	if len(requestBody) > 0 {
		if err := json.Unmarshal(requestBody, &body); err != nil {
			return nil, fmt.Errorf("invalid request body: %w", err)
		}
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
		return ar.Result == resultError ||
			ar.Result == resultFailure ||
			ar.Result == resultPartialSuccess
	})
}
