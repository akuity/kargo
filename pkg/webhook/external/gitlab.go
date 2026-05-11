package external

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

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
		case gl.EventTypeMergeRequest, gl.EventTypePush, gl.EventTypeTagPush:
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
		case *gl.MergeEvent:
			g.handleMergeRequestEvent(ctx, w, e)
		case *gl.PushEvent:
			g.handlePushEvent(ctx, w, e)
		case *gl.TagEvent:
			g.handleTagPushEvent(ctx, w, e)
		}
	})
}

func (g *gitlabWebhookReceiver) handleMergeRequestEvent(
	ctx context.Context,
	w http.ResponseWriter,
	e *gl.MergeEvent,
) {
	action := e.ObjectAttributes.Action
	if action != "close" && action != "merge" {
		xhttp.WriteResponseJSON(w, http.StatusOK, nil)
		return
	}
	prURL := e.ObjectAttributes.URL
	logger := logging.LoggerFromContext(ctx)
	logger = logger.WithValues("prURL", prURL)
	ctx = logging.ContextWithLogger(ctx, logger)
	refreshPromotionsByPrURL(ctx, w, g.client, g.project, prURL)
}

func (g *gitlabWebhookReceiver) handlePushEvent(
	ctx context.Context,
	w http.ResponseWriter,
	e *gl.PushEvent,
) {
	logger := logging.LoggerFromContext(ctx)

	var repoURLs []string
	if e.Repository != nil {
		repoURLs = []string{
			urls.NormalizeGit(e.Repository.GitHTTPURL),
			urls.NormalizeGit(e.Repository.GitSSHURL),
		}
	}
	var changedFiles []string
	if e.TotalCommitsCount > int64(len(e.Commits)) {
		logger.Info(
			"push event commits were truncated by GitLab; "+
				"skipping path filtering for this event",
			"totalCommits", e.TotalCommitsCount,
			"receivedCommits", len(e.Commits),
		)
	} else {
		changedFiles = collectPaths(e.Commits, func(c *gl.PushEventCommit) []string {
			return slices.Concat(c.Added, c.Modified, c.Removed)
		})
	}
	logger = logger.WithValues(
		"repoURLs", repoURLs,
		"ref", e.Ref,
	)
	ctx = logging.ContextWithLogger(ctx, logger)
	refreshWarehouses(ctx, w, g.client, g.project, repoURLs, changedFiles, e.Ref)
}

func (g *gitlabWebhookReceiver) handleTagPushEvent(
	ctx context.Context,
	w http.ResponseWriter,
	e *gl.TagEvent,
) {
	var repoURLs []string
	if e.Repository != nil {
		repoURLs = []string{
			urls.NormalizeGit(e.Repository.GitHTTPURL),
			urls.NormalizeGit(e.Repository.GitSSHURL),
		}
	}
	logger := logging.LoggerFromContext(ctx)
	logger = logger.WithValues(
		"repoURLs", repoURLs,
		"tag", e.Ref,
	)
	ctx = logging.ContextWithLogger(ctx, logger)
	refreshWarehouses(ctx, w, g.client, g.project, repoURLs, nil, e.Ref)
}
