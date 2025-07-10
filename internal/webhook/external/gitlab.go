package external

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	gl "gitlab.com/gitlab-org/api/client-go"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libGit "github.com/akuity/kargo/internal/controller/git"
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
			rc := &refreshEligibilityChecker{
				git: newGitlabCodeChange(e),
			}
			refreshWarehouses(ctx, w, g.client, g.project, rc, repoURL)
		}
	})
}

func newGitlabCodeChange(e *gl.PushEvent) *codeChange {
	hc := e.Commits[len(e.Commits)-1] // last commit is the newest

	// expected format "Name <email>".
	author := fmt.Sprintf("%s <%s>",
		hc.Author.Name,
		hc.Author.Email,
	)

	var createdAt time.Time
	if hc.Timestamp != nil {
		createdAt = *hc.Timestamp
	}

	var diffs []string
	if hc != nil {
		if len(hc.Added) > 0 {
			diffs = append(diffs, hc.Added...)
		}
		if len(hc.Modified) > 0 {
			diffs = append(diffs, hc.Modified...)
		}
		if len(hc.Removed) > 0 {
			diffs = append(diffs, hc.Removed...)
		}
	}

	return &codeChange{
		tag: &libGit.TagMetadata{
			Tag:         strings.TrimPrefix(e.Ref, "refs/tags/"),
			CommitID:    hc.ID,
			CreatorDate: createdAt,
			Author:      author,
			Committer:   author,
			Subject:     hc.Message,
			Tagger:      author,
		},
		commit: &libGit.CommitMetadata{
			ID:         hc.ID,
			CommitDate: createdAt,
			Author:     author,
			Committer:  author,
			Subject:    hc.Message,
		},
		branch: strings.TrimPrefix(e.Ref, "refs/heads/"),
		diffs:  diffs,
	}
}
