package external

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/helm"
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
			image.NormalizeURL(payload.DockerURL),
			helm.NormalizeChartRepositoryURL(payload.DockerURL),
		}

		logger = logger.WithValues("repoURLs", repoURLs)
		ctx = logging.ContextWithLogger(ctx, logger)
		newestTag := strPtr(payload.UpdatedTags[len(payload.UpdatedTags)-1])
		rc := &refreshEligibilityChecker{
			newImageTag: newestTag,
			newChartTag: newestTag,
		}
		refreshWarehouses(ctx, w, q.client, q.project, rc, repoURLs...)
	})
}
