package external

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/image"
	"github.com/akuity/kargo/internal/io"
	"github.com/akuity/kargo/internal/logging"
)

const (
	dockerhub                    = "dockerhub"
	dockerhubSecretDataKey       = "secret"
	dockerhubWebhookBodyMaxBytes = 2 << 20 // 2MB
)

func init() {
	registry.register(
		dockerhub,
		webhookReceiverRegistration{
			predicate: func(cfg kargoapi.WebhookReceiverConfig) bool {
				return cfg.DockerHub != nil
			},
			factory: newDockerHubWebhookReceiver,
		},
	)
}

// dockerhubWebhookReceiver is an implementation of WebhookReceiver that handles
// inbound webhooks from Docker Hub.
type dockerhubWebhookReceiver struct {
	*baseWebhookReceiver
}

// newDockerHubWebhookReceiver returns a new instance of
// dockerhubWebhookReceiver.
func newDockerHubWebhookReceiver(
	c client.Client,
	project string,
	cfg kargoapi.WebhookReceiverConfig,
) WebhookReceiver {
	return &dockerhubWebhookReceiver{
		baseWebhookReceiver: &baseWebhookReceiver{
			client:     c,
			project:    project,
			secretName: cfg.GitHub.SecretRef.Name,
		},
	}
}

// getReceiverType implements WebhookReceiver.
func (d *dockerhubWebhookReceiver) getReceiverType() string {
	return dockerhub
}

// getSecretValues implements WebhookReceiver.
func (d *dockerhubWebhookReceiver) getSecretValues(
	secretData map[string][]byte,
) ([]string, error) {
	secretValue, ok := secretData[dockerhubSecretDataKey]
	if !ok {
		return nil,
			errors.New("Secret data is not valid for a Docker Hub WebhookReceiver")
	}
	return []string{string(secretValue)}, nil
}

// GetHandler implements WebhookReceiver.
func (d *dockerhubWebhookReceiver) GetHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := logging.LoggerFromContext(ctx)

		// Early check of Content-Length if available
		if contentLength := r.ContentLength; contentLength > dockerhubWebhookBodyMaxBytes {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("content exceeds limit of %d bytes", dockerhubWebhookBodyMaxBytes),
					http.StatusRequestEntityTooLarge,
				),
			)
			return
		}

		body, err := io.LimitRead(r.Body, dockerhubWebhookBodyMaxBytes)
		if err != nil {
			if errors.Is(err, &io.BodyTooLargeError{}) {
				xhttp.WriteErrorJSON(
					w,
					xhttp.Error(err, http.StatusRequestEntityTooLarge),
				)
				return
			}
			xhttp.WriteErrorJSON(w, err)
			return
		}

		payload := struct {
			Repository struct {
				RepoName string `json:"repo_name"`
			} `json:"repository"`
		}{}

		if err = json.Unmarshal(body, &payload); err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("invalid request body"), http.StatusBadRequest),
			)
			return
		}

		// Normalize the repo name
		repoURL := image.NormalizeURL(payload.Repository.RepoName)

		logger = logger.WithValues("repoURL", repoURL)
		ctx = logging.ContextWithLogger(ctx, logger)

		result, err := refreshWarehouses(ctx, d.client, d.project, repoURL)
		if err != nil {
			xhttp.WriteErrorJSON(w, err)
			return
		}
		if result.failures > 0 {
			xhttp.WriteResponseJSON(
				w,
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
		xhttp.WriteResponseJSON(
			w,
			http.StatusOK,
			map[string]string{
				"msg": fmt.Sprintf("refreshed %d warehouse(s)", result.successes),
			},
		)

	})
}
