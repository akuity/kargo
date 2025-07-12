package external

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	gl "gitlab.com/gitlab-org/api/client-go"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/git"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/logging"
)

const (
	gitlab              = "gitlab"
	gitLabSecretDataKey = "secret-token"
)

func init() {
	registry.register(
		gitlab,
		webhookReceiverRegistration{
			predicate: func(cfg kargoapi.WebhookReceiverConfig) bool {
				return cfg.GitLab != nil
			},
			factory: newGitLabWebhookReceiver,
		},
	)
}

// gitlabWebhookReceiver is an implementation of WebhookReceiver that handles
// inbound webhooks from GitLab.
type gitlabWebhookReceiver struct {
	*baseWebhookReceiver
}

// newGitLabWebhookReceiver returns a new instance of gitlabWebhookReceiver.
func newGitLabWebhookReceiver(
	c client.Client,
	project string,
	cfg kargoapi.WebhookReceiverConfig,
) WebhookReceiver {
	return &gitlabWebhookReceiver{
		baseWebhookReceiver: &baseWebhookReceiver{
			client:     c,
			project:    project,
			secretName: cfg.GitLab.SecretRef.Name,
		},
	}
}

// GetDetails implements WebhookReceiver.
func (g *gitlabWebhookReceiver) getReceiverType() string {
	return gitlab
}

// getSecretValues implements WebhookReceiver.
func (g *gitlabWebhookReceiver) getSecretValues(
	secretData map[string][]byte,
) ([]string, error) {
	token, ok := secretData[gitLabSecretDataKey]
	if !ok {
		return nil,
			errors.New("Secret data is not valid for a GitLab WebhookReceiver")
	}
	return []string{string(token)}, nil
}

// getHandler implements WebhookReceiver.
func (g *gitlabWebhookReceiver) getHandler(requestBody []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := logging.LoggerFromContext(ctx)

		token, ok := g.secretData[gitLabSecretDataKey]
		if !ok {
			xhttp.WriteErrorJSON(w, nil)
			return
		}

		if string(token) != gl.HookEventToken(r) {
			xhttp.WriteErrorJSON(w, xhttp.Error(nil, http.StatusUnauthorized))
			return
		}

		eventType := gl.HookEventType(r)
		switch eventType {
		case gl.EventTypePush:
		default:
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("event type %s is not supported", eventType),
					http.StatusNotImplemented,
				),
			)
			return
		}

		logger = logger.WithValues("eventType", eventType)
		ctx = logging.ContextWithLogger(ctx, logger)

		event, err := gl.ParseWebhook(eventType, requestBody)
		if err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("invalid request body"), http.StatusBadRequest),
			)
			return
		}

		switch e := event.(type) {
		case *gl.PushEvent:
			var repoURL string
			if e.Repository != nil {
				repoURL = git.NormalizeURL(e.Repository.GitHTTPURL)
			}
			logger = logger.WithValues("repoURL", repoURL)
			ctx = logging.ContextWithLogger(ctx, logger)
			rc := newGitLabRefreshCheck(e)
			refreshWarehouses(ctx, w, g.client, g.project, rc, repoURL)
		}
	})
}

// newGitLabRefreshCheck creates a new refreshEligibilityChecker from a GitLab push event.
// It extracts the metadata from the event payload. This is used downstream to determine which Warehouses
// should be refreshed in response to the event based on the commit selection
// strategy configured for the Warehouse.
//
// See the GitLab Push event payload documentation for more details:
//
//	https://docs.gitlab.com/user/project/integrations/webhook_events/#push-events
func newGitLabRefreshCheck(e *gl.PushEvent) *refreshEligibilityChecker {
	var branchName, tag *string
	if e.Ref != "" {
		switch {
		case strings.HasPrefix(e.Ref, "refs/tags/"):
			tag = strPtr(strings.TrimPrefix(e.Ref, "refs/tags/"))
		case strings.HasPrefix(e.Ref, "refs/heads/"):
			branchName = strPtr(strings.TrimPrefix(e.Ref, "refs/heads/"))
		}
	}
	return &refreshEligibilityChecker{
		newGitTag:  tag,
		branchName: branchName,
	}
}
