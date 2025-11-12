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
	xhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/urls"
)

const (
	azureSecretDataKey = "secret"
	azure              = "azure"

	acrPingEvent         = "ping"
	acrPushEvent         = "push"
	azureDevOpsPushEvent = "git.push"

	acrUserAgentPrefix         = "AzureContainerRegistry"
	azureDevOpsUserAgentPrefix = "VSServices"
)

func init() {
	defaultWebhookReceiverRegistry.MustRegister(
		webhookReceiverRegistration{
			Predicate: func(_ context.Context, cfg kargoapi.WebhookReceiverConfig) (bool, error) {
				return cfg.Azure != nil, nil
			},
			Value: newAzureWebhookReceiver,
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
		case strings.HasPrefix(r.UserAgent(), acrUserAgentPrefix):
			a.handleACREvent(ctx, w, requestBody)
		// Format is VSServices/<version>
		case strings.HasPrefix(r.UserAgent(), azureDevOpsUserAgentPrefix):
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
	case acrPushEvent:
		repoURLs := getNormalizedImageRepoURLs(
			fmt.Sprintf("%s/%s", event.Request.Host, event.Target.Repository),
			event.Target.MediaType,
		)
		logger := logging.LoggerFromContext(ctx).WithValues(
			"repoURLs", repoURLs,
			"mediaType", event.Target.MediaType,
			"tag", event.Target.Tag,
		)
		ctx = logging.ContextWithLogger(ctx, logger)
		refreshWarehouses(ctx, w, a.client, a.project, repoURLs, event.Target.Tag)
	case acrPingEvent:
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

	if event.EventType != azureDevOpsPushEvent {
		xhttp.WriteErrorJSON(
			w,
			xhttp.Error(
				fmt.Errorf("event type %s is not supported", event.EventType),
				http.StatusBadRequest,
			),
		)
		return
	}

	repoURLs := []string{urls.NormalizeGit(event.Resource.Repository.RemoteURL)}
	logger := logging.LoggerFromContext(ctx)
	refs := event.getRefs()
	logger = logger.WithValues(
		"repoURLs", repoURLs,
		"refs", refs,
	)
	ctx = logging.ContextWithLogger(ctx, logger)
	refreshWarehouses(ctx, w, a.client, a.project, repoURLs, refs...)
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
		Tag        string `json:"tag"`
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
		RefUpdates []struct {
			Name string `json:"name,omitempty"`
		} `json:"refUpdates,omitempty"`
		Repository struct {
			RemoteURL string `json:"remoteUrl,omitempty"`
		} `json:"repository"`
	} `json:"resource"`
}

// getRefs extracts all references mentioned by the event
func (event azureDevOpsEvent) getRefs() []string {
	var qualifiers []string
	for _, refUpdate := range event.Resource.RefUpdates {
		qualifiers = append(qualifiers, refUpdate.Name)
	}
	return qualifiers
}
