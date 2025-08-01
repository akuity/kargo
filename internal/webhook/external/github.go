package external

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	gh "github.com/google/go-github/v74/github"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/git"
	xhttp "github.com/akuity/kargo/internal/http"
	"github.com/akuity/kargo/internal/logging"
)

const (
	GithubSecretDataKey = "secret"

	github = "github"

	// githubEventTypePackage corresponds to a package push event when the webhook
	// has been registered directly at the repository-level. i.e. When the webhook
	// is not registered indirectly via a GitHub App that's been installed into
	// the repository.
	githubEventTypePackage = "package"
	githubEventTypePing    = "ping"
	githubEventTypePush    = "push"
	// githubEventTypeRegistryPackage corresponds to a package push event when the
	// webhook has been registered indirectly via a GitHub App that's been
	// installed into the repository. i.e. When the webhook has not been
	// registered directly at the repository-level.
	githubEventTypeRegistryPackage = "registry_package"

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
		return nil, fmt.Errorf("missing %q data key for GitHub WebhookReceiver", GithubSecretDataKey)
	}
	return []string{string(secretValue)}, nil
}

// getHandler implements WebhookReceiver.
func (g *githubWebhookReceiver) getHandler(requestBody []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger := logging.LoggerFromContext(ctx)

		signingKey, ok := g.secretData[GithubSecretDataKey] // a.k.a. shared secret
		if !ok {
			xhttp.WriteErrorJSON(w, nil)
			return
		}

		eventType := r.Header.Get(gh.EventTypeHeader)
		switch eventType {
		case githubEventTypePackage,
			githubEventTypePing,
			githubEventTypePush,
			githubEventTypeRegistryPackage:
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

		sig := r.Header.Get(gh.SHA256SignatureHeader)
		if sig == "" {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("missing signature"), http.StatusUnauthorized),
			)
			return
		}

		if err := gh.ValidateSignature(sig, requestBody, signingKey); err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("unauthorized"), http.StatusUnauthorized),
			)
			return
		}

		event, err := gh.ParseWebHook(eventType, requestBody)
		if err != nil {
			xhttp.WriteErrorJSON(
				w,
				xhttp.Error(errors.New("invalid request body"), http.StatusBadRequest),
			)
			return
		}

		var qualifiers []string
		var repoURLs []string
		var mediaType string

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
			v := pkg.GetPackageVersion()
			manifest := v.GetContainerMetadata().GetManifest()
			// Determine if the package is a Helm chart
			if cfg, ok := manifest["config"].(map[string]any); ok {
				mediaType, _ = cfg["media_type"].(string)
			}
			repoURLs = getNormalizedImageRepoURLs(
				pkg.GetPackageVersion().GetPackageURL(),
				mediaType,
			)
			tag := v.GetContainerMetadata().GetTag().GetName()
			qualifiers = []string{tag}
			logger = logger.WithValues("tag", tag)
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
			ref := e.GetRef()
			qualifiers = []string{ref}
			logger = logger.WithValues("ref", ref)
			// TODO(krancour): GetHTMLURL() gives us a repo URL starting with
			// https://. By refreshing Warehouses using a normalized representation of
			// that URL, we will miss any Warehouses that are subscribed to the same
			// repository using a different URL format.
			repoURLs = []string{git.NormalizeURL(e.GetRepo().GetCloneURL())}

		case *gh.RegistryPackageEvent:
			switch e.GetAction() {
			// These are the only actions that should refresh Warehouses.
			case "published", "updated":
			default:
				xhttp.WriteResponseJSON(w, http.StatusOK, nil)
				return
			}
			pkg := e.GetRegistryPackage()
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
			// GitHub sometimes sends package URLs with a trailing colon and no tag
			// (e.g., "ghcr.io/user/image:"). Such strings are not valid OCI image
			// references. We trim the trailing colon to avoid parsing errors.
			pkgURL := strings.TrimSuffix(pkg.GetPackageVersion().GetPackageURL(), ":")
			manifest := pkg.GetPackageVersion().GetContainerMetadata().GetManifest()
			if cfg, ok := manifest["config"].(map[string]any); ok {
				if mediaType, ok = cfg["media_type"].(string); ok {
					repoURLs = getNormalizedImageRepoURLs(pkgURL, mediaType)
				}
			}
		}

		logger = logger.WithValues(
			"repoURLs", repoURLs,
			"mediaType", mediaType,
		)
		ctx = logging.ContextWithLogger(ctx, logger)

		refreshWarehouses(ctx, w, g.client, g.project, repoURLs, qualifiers...)
	})
}
