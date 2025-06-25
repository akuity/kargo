package external

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/helm"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/image"
	"github.com/akuity/kargo/internal/logging"
)

const (
	artifactorySecretDataKey = "secret-token"
	artifactory              = "artifactory"
)

func init() {
	registry.register(
		artifactory,
		webhookReceiverRegistration{
			predicate: func(cfg kargoapi.WebhookReceiverConfig) bool {
				return cfg.Artifactory != nil
			},
			factory: newArtifactoryWebhookReceiver,
		},
	)
}

// artifactoryWebhookReceiver is an implementation of WebhookReceiver that handles
// inbound webhooks from Jfrog Artifactory.
type artifactoryWebhookReceiver struct {
	*baseWebhookReceiver
}

// newArtifactoryWebhookReceiver returns a new instance of artifactoryWebhookReceiver.
func newArtifactoryWebhookReceiver(
	c client.Client,
	project string,
	cfg kargoapi.WebhookReceiverConfig,
) WebhookReceiver {
	return &artifactoryWebhookReceiver{
		baseWebhookReceiver: &baseWebhookReceiver{
			client:     c,
			project:    project,
			secretName: cfg.Artifactory.SecretRef.Name,
		},
	}
}

// GetDetails implements WebhookReceiver.
func (a *artifactoryWebhookReceiver) getReceiverType() string {
	return artifactory
}

// getSecretValues implements WebhookReceiver.
func (a *artifactoryWebhookReceiver) getSecretValues(
	secretData map[string][]byte,
) ([]string, error) {
	secretValue, ok := secretData[artifactorySecretDataKey]
	if !ok {
		return nil,
			errors.New("Secret data is not valid for a Artifactory WebhookReceiver")
	}
	return []string{string(secretValue)}, nil
}

// getHandler implements WebhookReceiver.
func (a *artifactoryWebhookReceiver) getHandler(requestBody []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.LoggerFromContext(ctx)

		token, ok := a.secretData[artifactorySecretDataKey]
		if !ok {
			xhttp.WriteErrorJSON(w, nil)
			return
		}

		sig := r.Header.Get("x-jfrog-event-auth")
		if sig == "" {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					errors.New("missing signature"),
					http.StatusUnauthorized,
				),
			)
			return
		}

		mac := hmac.New(sha256.New, token)
		mac.Write(requestBody)
		computedSig := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(sig), []byte(computedSig)) {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					errors.New("unauthorized"),
					http.StatusUnauthorized,
				),
			)
			return
		}

		logger.Info("Received Artifactory webhook",
			"raw event", string(requestBody),
		)

		var payload artifactoryPushEvent
		if err := json.Unmarshal(requestBody, &payload); err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("invalid request body"),
					http.StatusBadRequest,
				),
			)
			return
		}

		switch payload.EventType {
		case "pushed":
			repoURL, err := url.JoinPath(
				strings.TrimPrefix(payload.Origin, "https://"),
				payload.Data.RepoKey,
				payload.Data.ImageName,
			)
			if err != nil {
				xhttp.WriteErrorJSON(
					w,
					xhttp.Error(
						fmt.Errorf("failed to construct repository URL: %w", err),
						http.StatusBadRequest,
					),
				)
				return
			}

			logger = logger.WithValues("repoURL", repoURL)
			ctx = logging.ContextWithLogger(ctx, logger)

			// Payloads from Artifactory contain no information about media type, so we
			// normalize the URL BOTH as if it were an image repo URL and as if it were
			// a chart repository URL. These will coincidentally be the same, but by
			// doing this, we safeguard against future changes to normalization logic.
			// Note: The refresh logic will dedupe the URLs, so this does not create
			// the possibility of a double refresh.
			repoURLs := []string{
				image.NormalizeURL(repoURL),
				helm.NormalizeChartRepositoryURL(repoURL),
			}
			refreshWarehouses(ctx, w, a.client, a.project, repoURLs...)
		default:
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("event type %s is not supported", payload.EventType),
					http.StatusNotImplemented,
				),
			)
			return
		}
	})
}

type artifactoryPushEvent struct {
	EventType string                   `json:"event_type"`
	Data      artifactoryPushEventData `json:"data"`
	Origin    string                   `json:"jpd_origin"`
}

type artifactoryPushEventData struct {
	RepoKey   string `json:"repo_key"`
	ImageName string `json:"image_name"`
}
