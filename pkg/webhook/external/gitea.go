package external

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	gh "github.com/google/go-github/v76/github"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	xhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/urls"
)

const (
	gitea = "gitea"

	giteaEventTypeHeader = "X-Gitea-Event"
	giteaSignatureHeader = "X-Hub-Signature-256"
	giteaSecretDataKey   = "secret"

	giteaEventTypePush = "push"
)

func init() {
	defaultWebhookReceiverRegistry.MustRegister(
		webhookReceiverRegistration{
			Predicate: func(_ context.Context, cfg kargoapi.WebhookReceiverConfig) (bool, error) {
				return cfg.Gitea != nil, nil
			},
			Value: newGiteaWebhookReceiver,
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
	secretValue, ok := secretData[giteaSecretDataKey]
	if !ok {
		return nil,
			errors.New("secret data is not valid for a Gitea WebhookReceiver")
	}
	return []string{string(secretValue)}, nil
}

// getHandler implements WebhookReceiver.
func (g *giteaWebhookReceiver) getHandler(requestBody []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := logging.LoggerFromContext(ctx)

		signingKey, ok := g.secretData[giteaSecretDataKey] // a.k.a. shared secret
		if !ok {
			xhttp.WriteErrorJSON(w, nil)
			return
		}

		eventType := r.Header.Get(giteaEventTypeHeader)
		switch eventType {
		case giteaEventTypePush:
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

		sig := r.Header.Get(giteaSignatureHeader)
		if sig == "" {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("missing signature"), http.StatusUnauthorized),
			)
			return
		}

		// Note: github.com/google/go-github/v76/github has a great implementation
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
			Ref  string `json:"ref"`
			Repo struct {
				URL string `json:"clone_url"`
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
		repoURLs := []string{urls.NormalizeGit(payload.Repo.URL)}

		logger = logger.WithValues(
			"repoURLs", repoURLs,
			"ref", payload.Ref,
		)
		ctx = logging.ContextWithLogger(ctx, logger)
		refreshWarehouses(ctx, w, g.client, g.project, repoURLs, payload.Ref)
	})
}
