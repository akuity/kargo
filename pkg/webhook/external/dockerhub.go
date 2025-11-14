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
)

const (
	dockerhub              = "dockerhub"
	dockerhubSecretDataKey = "secret"
)

func init() {
	defaultWebhookReceiverRegistry.MustRegister(
		webhookReceiverRegistration{
			Predicate: func(_ context.Context, cfg kargoapi.WebhookReceiverConfig) (bool, error) {
				return cfg.DockerHub != nil, nil
			},
			Value: newDockerHubWebhookReceiver,
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
			secretName: cfg.DockerHub.SecretRef.Name,
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
		return nil, fmt.Errorf("missing data key %q for DockerHub WebhookReceiver", dockerhubSecretDataKey)
	}
	return []string{string(secretValue)}, nil
}

// getHandler implements WebhookReceiver.
func (d *dockerhubWebhookReceiver) getHandler(requestBody []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := logging.LoggerFromContext(ctx)

		payload := struct {
			PushData struct {
				MediaType string `json:"media_type"`
				Tag       string `json:"tag"`
			} `json:"push_data"`
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

		repoURLs := getNormalizedImageRepoURLs(
			payload.Repository.RepoName,
			payload.PushData.MediaType,
		)
		logger = logger.WithValues(
			"repoURLs", repoURLs,
			"tag", payload.PushData.Tag,
			"mediaType", payload.PushData.MediaType,
		)
		ctx = logging.ContextWithLogger(ctx, logger)
		refreshWarehouses(ctx, w, d.client, d.project, repoURLs, payload.PushData.Tag)
	})
}
