package external

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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
// inbound webhooks from azure.
type azureWebhookReceiver struct {
	*baseWebhookReceiver
}

// newazureWebhookReceiver returns a new instance of azureWebhookReceiver.
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

// GetDetails implements WebhookReceiver.
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
		logger.Debug("identifying source repository")

		logger.Info("received Azure webhook event",
			"requestBody", string(requestBody),
		)

		var event azureEvent
		if err := json.Unmarshal(requestBody, &event); err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("invalid request body"),
					http.StatusBadRequest,
				),
			)
			return
		}

		switch {
		case event.Action == "ping":
			xhttp.WriteResponseJSON(
				w,
				http.StatusOK,
				map[string]string{
					"msg": "ping event received, webhook is configured correctly",
				},
			)
			return
		case event.Action == "push":
			repoURL := fmt.Sprintf("%s/%s",
				event.Request.Host,
				event.Target.Repository,
			)

			// Payloads from azure contain no information about media type, so we
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
			// For Azure DevOps, when you configure an action trigger
			// and test it through their UI, it sends an event type
			// that is the same as the one you configured it for
			// as opposed to a "ping" event like Azure Container Registry does.
		case event.EventType == "git.push":
			repoURL := git.NormalizeURL(event.Resource.Repository.RemoteURL)
			logger = logger.WithValues("repoURLs", repoURL)
			ctx = logging.ContextWithLogger(ctx, logger)
			refreshWarehouses(ctx, w, q.client, q.project, repoURL)
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
	})
}

// events can come from Azure Container Registry or Azure DevOps.
// azureEvent is a struct that represents the payload of an event from either
// of these services as there are no overlapping fields between the two
//
// For information on payload schemas for each service, see the links below:
//
//	Azure Container Registry
//		https://learn.microsoft.com/en-us/azure/container-registry/container-registry-webhook-reference#payload-example-image-push-event
//
//	Azure DevOps
//		https://learn.microsoft.com/en-us/azure/devops/service-hooks/services/webhooks?view=azure-devops#resource-details-to-send
//
// Note: there are no overlapping fields between the two services, so we can
// use a single struct to represent both payloads.
type azureEvent struct {
	Action    string `json:"action"`              // For Azure Container Registry
	EventType string `json:"eventType,omitempty"` // For Azure DevOps
	Target    struct {
		Repository string `json:"repository"`
	} `json:"target"` // For Azure Container Registry
	Request struct {
		Host string `json:"host"`
	} `json:"request"` // For Azure Container Registry
	Resource struct {
		Repository struct {
			RemoteURL string `json:"remoteUrl,omitempty"`
		} `json:"repository"`
	} `json:"resource"` // For Azure DevOps
}