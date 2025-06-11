package external

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/io"
	"github.com/akuity/kargo/internal/logging"
)

const (
	quaySecretDataKey       = "secret"
	quay                    = "quay"
	quayWebhookBodyMaxBytes = 2 << 20 // 2MB
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

// GetHandler implements WebhookReceiver.
func (q *quayWebhookReceiver) GetHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.LoggerFromContext(ctx)
		logger.Debug("identifying source repository")

		if contentLength := r.ContentLength; contentLength > quayWebhookBodyMaxBytes {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("content exceeds limit of %d bytes", quayWebhookBodyMaxBytes),
					http.StatusRequestEntityTooLarge,
				),
			)
			return
		}

		b, err := io.LimitRead(r.Body, quayWebhookBodyMaxBytes)
		if err != nil {
			if errors.Is(err, &io.BodyTooLargeError{}) {
				xhttp.WriteErrorJSON(
					w,
					xhttp.Error(err, http.StatusRequestEntityTooLarge),
				)
				return
			}
			xhttp.WriteErrorJSON(w, err)
		}

		payload := struct {
			// format: quay.io/mynamespace/repository
			DockerURL string `json:"docker_url"`
		}{}

		if err = json.Unmarshal(b, &payload); err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("invalid request body"),
					http.StatusBadRequest,
				),
			)
		}

		if payload.DockerURL == "" {
			xhttp.WriteErrorJSON(w,
				xhttp.Error(
					fmt.Errorf("missing repository web URL in request body"),
					http.StatusBadRequest,
				),
			)
			return
		}

		repoURL := payload.DockerURL

		logger = logger.WithValues("repoURL", repoURL)
		ctx = logging.ContextWithLogger(ctx, logger)

		result, err := refreshWarehouses(ctx, q.client, q.project, repoURL)
		if err != nil {
			xhttp.WriteErrorJSON(w, err)
			return
		}

		logger.Debug("execution complete",
			"successes", result.successes,
			"failures", result.failures,
		)

		if result.failures > 0 {
			xhttp.WriteResponseJSON(w,
				http.StatusInternalServerError,
				map[string]string{
					"error": fmt.Sprintf("failed to refresh %d of %d warehouses",
						result.failures,
						result.successes+result.failures,
					),
				},
			)
			return
		}

		xhttp.WriteResponseJSON(w,
			http.StatusOK,
			map[string]string{
				"msg": fmt.Sprintf("refreshed %d warehouse(s)",
					result.successes,
				),
			},
		)
	})
}
