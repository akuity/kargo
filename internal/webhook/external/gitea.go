package external

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	gh "github.com/google/go-github/v71/github"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/git"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/logging"
)

const (
	GiteaSecretDataKey = "secret"

	gitea = "gitea"
)

func init() {
	registry.register(
		gitea,
		webhookReceiverRegistration{
			predicate: func(cfg kargoapi.WebhookReceiverConfig) bool {
				return cfg.Gitea != nil
			},
			factory: newGiteaWebhookReceiver,
		},
	)
}

// giteaWebhookReceiver is an implementation of WebhookReceiver that handles
// inbound webhooks from Gitea.
type giteaWebhookReceiver struct {
	*baseWebhookReceiver
}

// newGiteaWebhookReceiver returns a new instance of giteaWebhookReceiver.
func newGiteaWebhookReceiver(
	c client.Client,
	project string,
	cfg kargoapi.WebhookReceiverConfig,
) WebhookReceiver {
	return &giteaWebhookReceiver{
		baseWebhookReceiver: &baseWebhookReceiver{
			client:     c,
			project:    project,
			secretName: cfg.Gitea.SecretRef.Name,
		},
	}
}

// getReceiverType implements WebhookReceiver.
func (g *giteaWebhookReceiver) getReceiverType() string {
	return gitea
}

// getSecretValues implements WebhookReceiver.
func (g *giteaWebhookReceiver) getSecretValues(
	secretData map[string][]byte,
) ([]string, error) {
	secretValue, ok := secretData[GiteaSecretDataKey]
	if !ok {
		return nil,
			errors.New("Secret data is not valid for a Gitea WebhookReceiver")
	}
	return []string{string(secretValue)}, nil
}

// getHandler implements WebhookReceiver.
func (g *giteaWebhookReceiver) getHandler(requestBody []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := logging.LoggerFromContext(ctx)

		signingKey, ok := g.secretData[GiteaSecretDataKey] // a.k.a. shared secret
		if !ok {
			xhttp.WriteErrorJSON(w, nil)
			return
		}

		eventType := r.Header.Get("X-Gitea-Event")
		switch eventType {
		case "ping", "push":
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

		sig := r.Header.Get("X-Gitea-Signature")
		if sig == "" {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("missing signature"), http.StatusUnauthorized),
			)
			return
		}

		// Note: github.com/google/go-github/v71/github has a great implementation
		// of HMAC signature validation that isn't GitHub-specific, so we've opted
		// to use it here for Gitea as well.
		if err := gh.ValidateSignature(sig, requestBody, signingKey); err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("unauthorized"), http.StatusUnauthorized),
			)
			return
		}

		payload := struct {
			Repo struct {
				Name string `json:"full_name"`
			} `json:"repository"`
		}{}
		if err := json.Unmarshal(requestBody, &payload); err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("invalid request body"), http.StatusBadRequest),
			)
			return
		}

		// Normalize the repo name
		repoURL := git.NormalizeURL(payload.Repo.Name)

		logger = logger.WithValues("repoURL", repoURL)
		ctx = logging.ContextWithLogger(ctx, logger)

		refreshWarehouses(ctx, w, g.client, g.project, repoURL)
	})
}
