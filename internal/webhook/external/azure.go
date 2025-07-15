package external

import (
	"context"
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
func (a *azureWebhookReceiver) getReceiverType() string {
	return azure
}

// getSecretValues implements WebhookReceiver.
func (a *azureWebhookReceiver) getSecretValues(
	secretData map[string][]byte,
) ([]string, error) {
	secretValue, ok := secretData[azureSecretDataKey]
	if !ok {
		return nil,
			fmt.Errorf("missing data key %q for Azure WebhookReceiver", azureSecretDataKey)
	}
	return []string{string(secretValue)}, nil
}

// getHandler implements WebhookReceiver.
func (a *azureWebhookReceiver) getHandler(requestBody []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.LoggerFromContext(ctx)
		logger.Debug("received azure event", "userAgent", r.UserAgent())

		switch {
		// Format is AzureContainerRegistry/<version>
		case strings.Contains(r.UserAgent(), "AzureContainerRegistry"):
			a.handleAcrEvent(ctx, w, requestBody)
		// Format is VSServices/<version>
		case strings.Contains(r.UserAgent(), "VSServices"):
			a.handleAzureDevOpsEvent(ctx, w, requestBody)
		default:
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					errors.New(
						"request does not appear to have originated from a supported service",
					),
					http.StatusBadRequest,
				),
			)
			return
		}
	})
}

func (a *azureWebhookReceiver) handleACREvent(
	ctx context.Context,
	w http.ResponseWriter,
	requestBody []byte,
) {
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
		repoURLs := resolveAcrRepoURLs(event)
		logger := logging.LoggerFromContext(ctx)
		logger = logger.WithValues("repoURLs", repoURLs)
		ctx = logging.ContextWithLogger(ctx, logger)
		refreshWarehouses(ctx, w, a.client, a.project, repoURLs...)
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
				http.StatusBadRequest,
			),
		)
		return
	}
}

func (a *azureWebhookReceiver) handleAzureDevOpsEvent(
	ctx context.Context,
	w http.ResponseWriter,
	requestBody []byte,
) {
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
	logger := logging.LoggerFromContext(ctx)
	logger = logger.WithValues("repoURL", repoURL)
	ctx = logging.ContextWithLogger(ctx, logger)
	refreshWarehouses(ctx, w, a.client, a.project, repoURL)
}

func resolveAcrRepoURLs(e acrEvent) []string {
	repoURL := fmt.Sprintf("%s/%s", e.Request.Host, e.Target.Repository)
	var repoURLs []string
	switch e.Target.MediaType {
	case imageMediaType:
		repoURLs = append(repoURLs, image.NormalizeURL(repoURL))
	case helmChartMediaType:
		repoURLs = append(repoURLs, helm.NormalizeChartRepositoryURL(repoURL))
	default:
		repoURLs = append(repoURLs,
			image.NormalizeURL(repoURL),
			helm.NormalizeChartRepositoryURL(repoURL),
		)
	}
	return repoURLs
}

// acrEvent represents the payload for Azure Container Registry webhooks.
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

// azureDevOpsEvent represents the payload for Azure DevOps webhooks.
// For information on payload schemas for Azure DevOps, see:
//
//	Azure DevOps
//		https://learn.microsoft.com/en-us/azure/devops/service-hooks/services/webhooks?view=azure-devops#resource-details-to-send
//
// nolint:lll
type azureDevOpsEvent struct {
	EventType string `json:"eventType,omitempty"`
	Resource  struct {
		Repository struct {
			RemoteURL string `json:"remoteUrl,omitempty"`
		} `json:"repository"`
	} `json:"resource"`
}
