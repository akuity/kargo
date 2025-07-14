package external

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/git"
	"github.com/akuity/kargo/internal/helm"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/image"
	"github.com/akuity/kargo/internal/logging"
)

const (
	imageMediaType     = "application/vnd.docker.distribution.manifest.v2+json"
	azureSecretDataKey = "secret"
	azure              = "azure"
)

func init() {
	registry.register(
		azure,
		webhookReceiverRegistration{
			predicate: func(cfg kargoapi.WebhookReceiverConfig) bool {
				return cfg.Azure != nil
			},
			factory: newAzureWebhookReceiver,
		},
	)
}

// azureWebhookReceiver is an implementation of WebhookReceiver that handles
// inbound webhooks from both Azure Container Registry (ACR) and Azure DevOps.
type azureWebhookReceiver struct {
	*baseWebhookReceiver
}

// newAzureWebhookReceiver returns a new instance of azureWebhookReceiver.
func newAzureWebhookReceiver(
	c client.Client,
	project string,
	cfg kargoapi.WebhookReceiverConfig,
) WebhookReceiver {
	return &azureWebhookReceiver{
		baseWebhookReceiver: &baseWebhookReceiver{
			client:     c,
			project:    project,
			secretName: cfg.Azure.SecretRef.Name,
		},
	}
}

// getReceiverType implements WebhookReceiver.
func (q *azureWebhookReceiver) getReceiverType() string {
	return azure
}

// getSecretValues implements WebhookReceiver.
func (q *azureWebhookReceiver) getSecretValues(
	secretData map[string][]byte,
) ([]string, error) {
	secretValue, ok := secretData[azureSecretDataKey]
	if !ok {
		return nil,
			errors.New("Secret data is not valid for an Azure WebhookReceiver")
	}
	return []string{string(secretValue)}, nil
}

// getHandler implements WebhookReceiver.
func (q *azureWebhookReceiver) getHandler(requestBody []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.LoggerFromContext(ctx)
		logger.Debug("received azure event", "userAgent", r.UserAgent())

		switch {
		// format is AzureContainerRegistry/<version>
		case strings.Contains(r.UserAgent(), "AzureContainerRegistry"):
			var event acrEvent
			if err := json.Unmarshal(requestBody, &event); err != nil {
				xhttp.WriteErrorJSON(
					w,
					xhttp.Error(errors.New("invalid request body"),
						http.StatusBadRequest,
					),
				)
				return
			}

			switch event.Action {
			case "push":
				repoURL := fmt.Sprintf("%s/%s",
					event.Request.Host,
					event.Target.Repository,
				)
				if event.Target.MediaType == imageMediaType {
					repoURL = image.NormalizeURL(repoURL)
				} else {
					repoURL = helm.NormalizeChartRepositoryURL(repoURL)
				}
				logger = logger.WithValues("repoURL", repoURL)
				ctx = logging.ContextWithLogger(ctx, logger)
				refreshWarehouses(ctx, w, q.client, q.project, repoURL)
			case "ping":
				xhttp.WriteResponseJSON(
					w,
					http.StatusOK,
					map[string]string{
						"msg": "ping event received, webhook is configured correctly",
					},
				)
			default:
				xhttp.WriteErrorJSON(
					w,
					xhttp.Error(
						fmt.Errorf("event type %s is not supported", event.Action),
						http.StatusNotImplemented,
					),
				)
				return
			}
		// Format is VSServices/<version>
		case strings.Contains(r.UserAgent(), "VSServices"):
			var event azureDevOpsEvent
			if err := json.Unmarshal(requestBody, &event); err != nil {
				xhttp.WriteErrorJSON(
					w,
					xhttp.Error(errors.New("invalid request body"),
						http.StatusBadRequest,
					),
				)
				return
			}
			repoURL := git.NormalizeURL(event.Resource.Repository.RemoteURL)
			logger = logger.WithValues("repoURL", repoURL)
			ctx = logging.ContextWithLogger(ctx, logger)
			refreshWarehouses(ctx, w, q.client, q.project, repoURL)
		default:
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("user agent %s is not supported", r.UserAgent()),
					http.StatusNotImplemented,
				),
			)
			return
		}
	})
}

// For more information on the payload schema for Azure Container Registry, see:
//
//	Azure Container Registry
//		https://learn.microsoft.com/en-us/azure/container-registry/container-registry-webhook-reference#payload-example-image-push-event
//
// nolint:lll
type acrEvent struct {
	Action string `json:"action"`
	Target struct {
		MediaType  string `json:"mediaType"`
		Repository string `json:"repository"`
	} `json:"target"`
	Request struct {
		Host string `json:"host"`
	} `json:"request"`
}

// For information on payload schemas for Azure DevOps, see:
//
//	Azure DevOps
//		https://learn.microsoft.com/en-us/azure/devops/service-hooks/services/webhooks?view=azure-devops#resource-details-to-send
//
// nolint:lll
type azureDevOpsEvent struct {
	EventType string `json:"eventType,omitempty"` // For Azure DevOps
	Resource  struct {
		Repository struct {
			RemoteURL string `json:"remoteUrl,omitempty"`
		} `json:"repository"`
	} `json:"resource"`
}
