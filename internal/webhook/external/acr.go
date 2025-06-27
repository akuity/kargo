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
	acrSecretDataKey = "secret"
	acr              = "acr"
)

func init() {
	registry.register(
		acr,
		webhookReceiverRegistration{
			predicate: func(cfg kargoapi.WebhookReceiverConfig) bool {
				return cfg.ACR != nil
			},
			factory: newACRWebhookReceiver,
		},
	)
}

// acrWebhookReceiver is an implementation of WebhookReceiver that handles
// inbound webhooks from acr.
type acrWebhookReceiver struct {
	*baseWebhookReceiver
}

// newACRWebhookReceiver returns a new instance of acrWebhookReceiver.
func newACRWebhookReceiver(
	c client.Client,
	project string,
	cfg kargoapi.WebhookReceiverConfig,
) WebhookReceiver {
	return &acrWebhookReceiver{
		baseWebhookReceiver: &baseWebhookReceiver{
			client:     c,
			project:    project,
			secretName: cfg.ACR.SecretRef.Name,
		},
	}
}

// GetDetails implements WebhookReceiver.
func (q *acrWebhookReceiver) getReceiverType() string {
	return acr
}

// getSecretValues implements WebhookReceiver.
func (q *acrWebhookReceiver) getSecretValues(
	secretData map[string][]byte,
) ([]string, error) {
	secretValue, ok := secretData[acrSecretDataKey]
	if !ok {
		return nil,
			errors.New("Secret data is not valid for an ACR WebhookReceiver")
	}
	return []string{string(secretValue)}, nil
}

// getHandler implements WebhookReceiver.
func (q *acrWebhookReceiver) getHandler(requestBody []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.LoggerFromContext(ctx)
		logger.Debug("identifying source repository")

		payload := struct {
			Action string `json:"action"`
			Target struct {
				Repository string `json:"repository"`
			} `json:"target"`
			Request struct {
				Host string `json:"host"`
			} `json:"request"`
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

		switch payload.Action {
		case "ping":
			xhttp.WriteResponseJSON(
				w,
				http.StatusOK,
				map[string]string{
					"msg": "ping event received, webhook is configured correctly",
				},
			)
			return
		case "push":
			repoURL := fmt.Sprintf("%s/%s",
				payload.Request.Host,
				payload.Target.Repository,
			)

			// Payloads from acr contain no information about media type, so we
			// normalize the URL BOTH as if it were an image repo URL and as if it were
			// a chart repository URL. These will coincidentally be the same, but by
			// doing this, we safeguard against future changes to normalization logic.
			// Note: The refresh logic will dedupe the URLs, so this does not create
			// the possibility of a double refresh.
			repoURLs := []string{
				image.NormalizeURL(repoURL),
				helm.NormalizeChartRepositoryURL(repoURL),
			}

			logger = logger.WithValues("repoURLs", repoURLs)
			ctx = logging.ContextWithLogger(ctx, logger)

			refreshWarehouses(ctx, w, q.client, q.project, repoURL)
		default:
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("event type %s is not supported", payload.Action),
					http.StatusNotImplemented,
				),
			)
			return
		}
	})
}
