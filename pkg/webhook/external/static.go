package external

import (
	"errors"
	"fmt"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	xhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
)

const static = "static"
const staticSecretDataKey = "secret-token"

func init() {
	registry.register(
		static,
		webhookReceiverRegistration{
			predicate: func(cfg kargoapi.WebhookReceiverConfig) bool {
				return cfg.Static != nil
			},
			factory: newStaticWebhookReceiver,
		},
	)
}

// staticWebhookReceiver is an implementation of WebhookReceiver that
// handles inbound webhooks from generic static sources.
type staticWebhookReceiver struct {
	*baseWebhookReceiver
	rule kargoapi.StaticWebhookRule
}

// newStaticWebhookReceiver returns a new instance of
// staticWebhookReceiver.
func newStaticWebhookReceiver(
	c client.Client,
	project string,
	cfg kargoapi.WebhookReceiverConfig,
) WebhookReceiver {
	return &staticWebhookReceiver{
		baseWebhookReceiver: &baseWebhookReceiver{
			client:     c,
			project:    project,
			secretName: cfg.Static.SecretRef.Name,
		},
		rule: cfg.Static.Rule,
	}
}

// getReceiverType implements WebhookReceiver.
func (s *staticWebhookReceiver) getReceiverType() string {
	return static
}

// getSecretValues implements WebhookReceiver.
func (s *staticWebhookReceiver) getSecretValues(
	secretData map[string][]byte,
) ([]string, error) {
	secretValue, ok := secretData[staticSecretDataKey]
	if !ok {
		return nil, errors.New("secret data is not valid for static WebhookReceiver")
	}
	return []string{string(secretValue)}, nil
}

// getHandler implements WebhookReceiver.
func (s *staticWebhookReceiver) getHandler(requestBody []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.LoggerFromContext(ctx)

		switch s.rule.Action {
		case kargoapi.StaticWebhookActionRefresh:
			if failures, err := refreshTargets(ctx, s.client, s.rule.Targets); err != nil {
				numSuccessful := len(s.rule.Targets) - failures
				logger.Info("finished refreshing targets with some failures",
					"numSuccessful", numSuccessful,
					"numFailed", failures,
				)
				xhttp.WriteErrorJSON(w,
					fmt.Errorf("%d out of %d targets failed to refresh",
						failures,
						len(s.rule.Targets),
					),
				)
				return
			}
			msg := fmt.Sprintf("successfully refreshed %d target(s)", len(s.rule.Targets))
			xhttp.WriteResponseJSON(w, http.StatusOK, map[string]string{"msg": msg})
		}
	})
}
