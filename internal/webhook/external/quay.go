package external

import (
	"encoding/json"
	"errors"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/image"
	"github.com/akuity/kargo/internal/logging"
)

const (
	quaySecretDataKey = "secret"
	quay              = "quay"
)

func init() {
	registry.register(
		quay,
		webhookReceiverRegistration{
			predicate: func(cfg kargoapi.WebhookReceiverConfig) bool {
				return cfg.Quay != nil
			},
			factory: newQuayWebhookReceiver,
		},
	)
}

// quayWebhookReceiver is an implementation of WebhookReceiver that handles
// inbound webhooks from Quay.
type quayWebhookReceiver struct {
	*baseWebhookReceiver
}

// newQuayWebhookReceiver returns a new instance of quayWebhookReceiver.
func newQuayWebhookReceiver(
	c client.Client,
	project string,
	cfg kargoapi.WebhookReceiverConfig,
) WebhookReceiver {
	return &quayWebhookReceiver{
		baseWebhookReceiver: &baseWebhookReceiver{
			client:     c,
			project:    project,
			secretName: cfg.Quay.SecretRef.Name,
		},
	}
}

// GetDetails implements WebhookReceiver.
func (q *quayWebhookReceiver) getReceiverType() string {
	return quay
}

// getSecretValues implements WebhookReceiver.
func (q *quayWebhookReceiver) getSecretValues(
	secretData map[string][]byte,
) ([]string, error) {
	secretValue, ok := secretData[quaySecretDataKey]
	if !ok {
		return nil,
			errors.New("Secret data is not valid for a Quay.io WebhookReceiver")
	}
	return []string{string(secretValue)}, nil
}

// getHandler implements WebhookReceiver.
func (q *quayWebhookReceiver) getHandler(requestBody []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.LoggerFromContext(ctx)
		logger.Debug("identifying source repository")

		payload := struct {
			// format: quay.io/mynamespace/repository
			DockerURL string `json:"docker_url"`
		}{}

		if err := json.Unmarshal(requestBody, &payload); err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("invalid request body"),
					http.StatusBadRequest,
				),
			)
			return
		}

		repoURL := image.NormalizeURL(payload.DockerURL)

		logger = logger.WithValues("repoURL", repoURL)
		ctx = logging.ContextWithLogger(ctx, logger)

		refreshWarehouses(ctx, w, q.client, q.project, repoURL)
	})
}
