package external

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	xhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/urls"
)

const (
	quaySecretDataKey = "secret"
	quay              = "quay"
)

func init() {
	defaultWebhookReceiverRegistry.MustRegister(
		webhookReceiverRegistration{
			Predicate: func(_ context.Context, cfg kargoapi.WebhookReceiverConfig) (bool, error) {
				return cfg.Quay != nil, nil
			},
			Value: newQuayWebhookReceiver,
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
		return nil, fmt.Errorf("missing %q data key for Quay.io WebhookReceiver", quaySecretDataKey)
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
			DockerURL   string   `json:"docker_url"`
			UpdatedTags []string `json:"updated_tags"`
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

		// Payloads from Quay contain no information about media type, so we
		// normalize the URL BOTH as if it were an image repo URL and as if it were
		// a chart repository URL. These will coincidentally be the same, but by
		// doing this, we safeguard against future changes to normalization logic.
		// Note: The refresh logic will dedupe the URLs, so this does not create
		// the possibility of a double refresh.
		repoURLs := []string{
			urls.NormalizeImage(payload.DockerURL),
			urls.NormalizeChart(payload.DockerURL),
		}

		logger = logger.WithValues(
			"repoURLs", repoURLs,
			"tags", payload.UpdatedTags,
		)
		ctx = logging.ContextWithLogger(ctx, logger)
		refreshWarehouses(ctx, w, q.client, q.project, repoURLs, payload.UpdatedTags...)
	})
}
