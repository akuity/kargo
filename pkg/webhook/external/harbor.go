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
	harborSecretDataKey = "auth-header"
	harborAuthHeader    = "Authorization"
	harbor              = "harbor"
	harborEventTypePush = "PUSH_ARTIFACT"
)

func init() {
	defaultWebhookReceiverRegistry.MustRegister(
		webhookReceiverRegistration{
			Predicate: func(_ context.Context, cfg kargoapi.WebhookReceiverConfig) (bool, error) {
				return cfg.Harbor != nil, nil
			},
			Value: newHarborWebhookReceiver,
		},
	)
}

// harborWebhookReceiver is an implementation of WebhookReceiver that handles
// inbound webhooks from Harbor.
type harborWebhookReceiver struct {
	*baseWebhookReceiver
}

// newHarborWebhookReceiver returns a new instance of harborWebhookReceiver.
func newHarborWebhookReceiver(
	c client.Client,
	project string,
	cfg kargoapi.WebhookReceiverConfig,
) WebhookReceiver {
	return &harborWebhookReceiver{
		baseWebhookReceiver: &baseWebhookReceiver{
			client:     c,
			project:    project,
			secretName: cfg.Harbor.SecretRef.Name,
		},
	}
}

// GetDetails implements WebhookReceiver.
func (h *harborWebhookReceiver) getReceiverType() string {
	return harbor
}

// getSecretValues implements WebhookReceiver.
func (h *harborWebhookReceiver) getSecretValues(
	secretData map[string][]byte,
) ([]string, error) {
	secretValue, ok := secretData[harborSecretDataKey]
	if !ok {
		return nil, fmt.Errorf(
			"missing data key %q for Harbor WebhookReceiver",
			harborSecretDataKey,
		)
	}
	return []string{string(secretValue)}, nil
}

// getHandler implements WebhookReceiver.
func (h *harborWebhookReceiver) getHandler(requestBody []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.LoggerFromContext(ctx)
		logger.Debug("identifying source repository")

		token, ok := h.secretData[harborSecretDataKey]
		if !ok {
			xhttp.WriteErrorJSON(w, nil)
			return
		}

		authHeader := r.Header.Get(harborAuthHeader)
		if authHeader == "" {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("missing authorization"), http.StatusUnauthorized),
			)
			return
		}

		// Harbor webhook authentication uses a simple string comparison, and is
		// expected to match whatever "Auth Header" was configured in Harbor's
		// webhook settings
		if authHeader != string(token) {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					errors.New("unauthorized"),
					http.StatusUnauthorized,
				),
			)
			return
		}

		// Note: this is Harbor's "Default" payload format, the "CloudEvents" format
		// is not supported
		payload := struct {
			Type      string `json:"type"`
			EventData struct {
				Resources []struct {
					Tag         string `json:"tag"`
					ResourceURL string `json:"resource_url"`
				} `json:"resources"`
			} `json:"event_data"`
		}{}

		logger.WithValues("payload", payload)

		if err := json.Unmarshal(requestBody, &payload); err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("invalid request body"),
					http.StatusBadRequest,
				),
			)
			return
		}

		// Only handle artifact push events for now
		if payload.Type != harborEventTypePush {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("unsupported event type"),
					http.StatusBadRequest,
				),
			)
			return
		}

		var repoURLs []string
		var tags []string
		if len(payload.EventData.Resources) > 0 {
			// Payloads from Harbor contain no information about media type, so we
			// normalize the URL BOTH as if it were an image repo URL and as if it
			// were a chart repository URL. These will coincidentally be the same, but
			// by doing this, we safeguard against future changes to normalization
			// logic. Note: The refresh logic will dedupe the URLs, so this does not
			// create the possibility of a double refresh.
			repoURLs = []string{
				urls.NormalizeImage(payload.EventData.Resources[0].ResourceURL),
				urls.NormalizeChart(payload.EventData.Resources[0].ResourceURL),
			}
			tags = make([]string, len(payload.EventData.Resources))
			for i, res := range payload.EventData.Resources {
				tags[i] = res.Tag
			}
		}

		logger = logger.WithValues(
			"repoURLs", repoURLs,
			"tags", tags,
		)
		ctx = logging.ContextWithLogger(ctx, logger)
		refreshWarehouses(ctx, w, h.client, h.project, repoURLs, tags...)
	})
}
