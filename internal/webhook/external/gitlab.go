package external

import (
	"errors"
	"fmt"
	"net/http"

	gl "gitlab.com/gitlab-org/api/client-go"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/git"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/io"
	"github.com/akuity/kargo/internal/logging"
)

const (
	gitlab                    = "gitlab"
	gitLabSecretDataKey       = "secret-token"
	gitlabWebhookBodyMaxBytes = 2 << 20 // 2MB
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

// GetHandler implements WebhookReceiver.
func (g *gitlabWebhookReceiver) GetHandler() http.HandlerFunc {
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

		// Early check of Content-Length if available
		if contentLength := r.ContentLength; contentLength > gitlabWebhookBodyMaxBytes {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("content exceeds limit of %d bytes", gitlabWebhookBodyMaxBytes),
					http.StatusRequestEntityTooLarge,
				),
			)
			return
		}

		body, err := io.LimitRead(r.Body, gitlabWebhookBodyMaxBytes)
		if err != nil {
			if errors.Is(err, &io.BodyTooLargeError{}) {
				xhttp.WriteErrorJSON(
					w,
					xhttp.Error(err, http.StatusRequestEntityTooLarge),
				)
				return
			}
			xhttp.WriteErrorJSON(w, err)
			return
		}

		event, err := gl.ParseWebhook(eventType, body)
		if err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("invalid request body"), http.StatusBadRequest),
			)
			return
		}

		switch e := event.(type) {
		case *gl.PushEvent:
			repoURL := git.NormalizeURL(e.Repository.GitHTTPURL)
			logger = logger.WithValues("repoURL", repoURL)
			ctx = logging.ContextWithLogger(ctx, logger)
			result, err := refreshWarehouses(ctx, g.client, g.project, repoURL)
			if err != nil {
				xhttp.WriteErrorJSON(w, err)
				return
			}
			if result.failures > 0 {
				xhttp.WriteResponseJSON(
					w,
					http.StatusInternalServerError,
					map[string]string{
						"error": fmt.Sprintf("failed to refresh %d of %d warehouses",
							result.failures,
							result.successes+result.failures,
						),
					},
				)
				return
			}
			xhttp.WriteResponseJSON(
				w,
				http.StatusOK,
				map[string]string{
					"msg": fmt.Sprintf("refreshed %d warehouse(s)", result.successes),
				},
			)
		}
	})
}
