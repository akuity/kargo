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
	dockerhub              = "dockerhub"
	dockerhubSecretDataKey = "secret"
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

// getHandler implements WebhookReceiver.
func (d *dockerhubWebhookReceiver) getHandler(requestBody []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := logging.LoggerFromContext(ctx)

		payload := struct {
			Repository struct {
				RepoName string `json:"repo_name"`
			} `json:"repository"`
		}{}

		if err := json.Unmarshal(requestBody, &payload); err != nil {
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

		refreshWarehouses(ctx, w, d.client, d.project, repoURL)
	})
}
