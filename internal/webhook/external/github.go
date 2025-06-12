package external

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/name"
	gh "github.com/google/go-github/v71/github"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/git"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/image"
	"github.com/akuity/kargo/internal/io"
	"github.com/akuity/kargo/internal/logging"
)

const (
	GithubSecretDataKey = "secret"

	github                    = "github"
	githubWebhookBodyMaxBytes = 2 << 20 // 2MB

	ghcrPackageTypeContainer = "CONTAINER"
	ghcrPackageTypeDocker    = "docker"
)

func init() {
	registry.register(
		github,
		webhookReceiverRegistration{
			predicate: func(cfg kargoapi.WebhookReceiverConfig) bool {
				return cfg.GitHub != nil
			},
			factory: newGitHubWebhookReceiver,
		},
	)
}

// githubWebhookReceiver is an implementation of WebhookReceiver that handles
// inbound webhooks from GitHub.
type githubWebhookReceiver struct {
	*baseWebhookReceiver
}

// newGitHubWebhookReceiver returns a new instance of githubWebhookReceiver.
func newGitHubWebhookReceiver(
	c client.Client,
	project string,
	cfg kargoapi.WebhookReceiverConfig,
) WebhookReceiver {
	return &githubWebhookReceiver{
		baseWebhookReceiver: &baseWebhookReceiver{
			client:     c,
			project:    project,
			secretName: cfg.GitHub.SecretRef.Name,
		},
	}
}

// getReceiverType implements WebhookReceiver.
func (g *githubWebhookReceiver) getReceiverType() string {
	return github
}

// getSecretValues implements WebhookReceiver.
func (g *githubWebhookReceiver) getSecretValues(
	secretData map[string][]byte,
) ([]string, error) {
	secretValue, ok := secretData[GithubSecretDataKey]
	if !ok {
		return nil,
			errors.New("Secret data is not valid for a GitHub WebhookReceiver")
	}
	return []string{string(secretValue)}, nil
}

// GetHandler implements WebhookReceiver.
func (g *githubWebhookReceiver) GetHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := logging.LoggerFromContext(ctx)

		signingKey, ok := g.secretData[GithubSecretDataKey] // a.k.a. shared secret
		if !ok {
			xhttp.WriteErrorJSON(w, nil)
			return
		}

		eventType := r.Header.Get("X-GitHub-Event")
		switch eventType {
		case "package", "ping", "push":
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
		if contentLength := r.ContentLength; contentLength > githubWebhookBodyMaxBytes {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(
					fmt.Errorf("content exceeds limit of %d bytes", githubWebhookBodyMaxBytes),
					http.StatusRequestEntityTooLarge,
				),
			)
			return
		}

		body, err := io.LimitRead(r.Body, githubWebhookBodyMaxBytes)
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

		sig := r.Header.Get(gh.SHA256SignatureHeader)
		if sig == "" {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("missing signature"), http.StatusUnauthorized),
			)
			return
		}

		if err = gh.ValidateSignature(sig, body, signingKey); err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("unauthorized"), http.StatusUnauthorized),
			)
			return
		}

		event, err := gh.ParseWebHook(eventType, body)
		if err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("invalid request body"), http.StatusBadRequest),
			)
			return
		}

		var repoURL string

		switch e := event.(type) {
		case *gh.PackageEvent:
			switch e.GetAction() {
			// These are the only actions that should refresh Warehouses.
			case "published", "updated":
			default:
				xhttp.WriteResponseJSON(w, http.StatusOK, nil)
				return
			}
			pkg := e.GetPackage()
			if pkg == nil || pkg.GetPackageVersion() == nil {
				xhttp.WriteErrorJSON(
					w,
					xhttp.Error(errors.New("invalid request body"), http.StatusBadRequest),
				)
				return
			}
			switch pkg.GetPackageType() {
			// These are the only types of packages we care about.
			case ghcrPackageTypeContainer, ghcrPackageTypeDocker:
			default:
				xhttp.WriteResponseJSON(w, http.StatusOK, nil)
				return
			}
			var ref name.Reference
			if ref, err = name.ParseReference(
				pkg.GetPackageVersion().GetPackageURL(),
			); err != nil {
				xhttp.WriteErrorJSON(w, err)
				return
			}
			repoURL = ref.Context().Name()
			// Only normalize if the registry is ghcr.io
			if ref.Context().Registry.Name() == "ghcr.io" {
				repoURL = image.NormalizeURL(repoURL)
			}

		case *gh.PingEvent:
			xhttp.WriteResponseJSON(
				w,
				http.StatusOK,
				map[string]string{
					"msg": "ping event received, webhook is configured correctly",
				},
			)
			return

		case *gh.PushEvent:
			// TODO(krancour): GetHTMLURL() gives use a repo URL starting with
			// https://. By refreshing Warehouses using a normalized representation of
			// that URL, we will miss any Warehouses that are subscribed to the same
			// repository using a different URL format.
			repoURL = git.NormalizeURL(e.GetRepo().GetCloneURL())
		}

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
	})
}
