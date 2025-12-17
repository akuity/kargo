package external

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	gl "gitlab.com/gitlab-org/api/client-go"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	xhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/urls"
)

const (
	gitlab = "gitlab"

	gitLabSecretDataKey = "secret-token"

	gitlabTokenHeader = "X-Gitlab-Token" // nolint: gosec
	gitlabEventHeader = "X-Gitlab-Event"
)

func init() {
	defaultWebhookReceiverRegistry.MustRegister(
		webhookReceiverRegistration{
			Predicate: func(_ context.Context, cfg kargoapi.WebhookReceiverConfig) (bool, error) {
				return cfg.GitLab != nil, nil
			},
			Value: newGitLabWebhookReceiver,
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
		return nil, fmt.Errorf("missing data key %q for GitLab WebhookReceiver", gitLabSecretDataKey)
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
		case gl.EventTypePush, gl.EventTypeTagPush:
		default:
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("event type %s is not supported", eventType),
					http.StatusBadRequest,
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
			var repoURLs []string
			if e.Repository != nil {
				repoURLs = []string{
					urls.NormalizeGit(e.Repository.GitHTTPURL),
					urls.NormalizeGit(e.Repository.GitSSHURL),
				}
			}
			logger = logger.WithValues(
				"repoURLs", repoURLs,
				"ref", e.Ref,
			)
			ctx = logging.ContextWithLogger(ctx, logger)
			refreshWarehouses(ctx, w, g.client, g.project, repoURLs, e.Ref)
		case *gl.TagEvent:
			var repoURLs []string
			if e.Repository != nil {
				repoURLs = []string{
					urls.NormalizeGit(e.Repository.GitHTTPURL),
					urls.NormalizeGit(e.Repository.GitSSHURL),
				}
			}
			logger = logger.WithValues(
				"repoURLs", repoURLs,
				"tag", e.Ref,
			)
			ctx = logging.ContextWithLogger(ctx, logger)
			refreshWarehouses(ctx, w, g.client, g.project, repoURLs, e.Ref)
		}
	})
}
