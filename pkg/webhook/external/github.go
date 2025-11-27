package external

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	gh "github.com/google/go-github/v76/github"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	xhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/urls"
)

const (
	GithubSecretDataKey = "secret"

	github = "github"

	// githubEventTypePackage corresponds to a package push event when the webhook
	// has been registered directly at the repository-level. i.e. When the webhook
	// has not been registered indirectly via a GitHub App that's been installed
	// into the repository.
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
	defaultWebhookReceiverRegistry.MustRegister(
		webhookReceiverRegistration{
			Predicate: func(_ context.Context, cfg kargoapi.WebhookReceiverConfig) (bool, error) {
				return cfg.GitHub != nil, nil
			},
			Value: newGitHubWebhookReceiver,
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

		switch e := event.(type) {
		case *gh.PackageEvent:
			action := e.GetAction()
			switch action {
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
			pkgVer := pkg.GetPackageVersion()
			containerMeta := pkgVer.GetContainerMetadata()
			manifest := containerMeta.GetManifest()
			var mediaType string
			if cfg, ok := manifest["config"].(map[string]any); ok {
				mediaType, _ = cfg["media_type"].(string)
			}
			repoURLs = getNormalizedImageRepoURLs(
				// In the case of `registry_package` events, we have sometimes observed
				// GitHub sending URLs with a trailing colon and no tag (e.g.,
				// "ghcr.io/user/image:"). Such strings are not valid OCI image
				// references. We've NOT seen this occur with `package` events, however,
				// the similarities between those two event types are so great that we
				// suspect it is a possibility. Out of an abundance of caution we are
				// trimming any trailing colon that may be present in this URL in order
				// to avoid parsing errors.
				strings.TrimSuffix(pkgVer.GetPackageURL(), ":"),
				mediaType,
			)

			tag := containerMeta.GetTag().GetName()
			qualifiers = []string{tag}
			logger = logger.WithValues(
				"mediaType", mediaType,
				"tag", tag,
			)
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
			repoURLs = []string{
				urls.NormalizeGit(e.GetRepo().GetCloneURL()),
				urls.NormalizeGit(e.GetRepo().GetSSHURL()),
			}
			ref := e.GetRef()
			qualifiers = []string{ref}
			logger = logger.WithValues("ref", ref)

		case *gh.RegistryPackageEvent:
			action := e.GetAction()
			switch action {
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
			pkgVer := pkg.GetPackageVersion()
			containerMeta := pkgVer.GetContainerMetadata()
			manifest := containerMeta.GetManifest()
			var mediaType string
			if cfg, ok := manifest["config"].(map[string]any); ok {
				mediaType, _ = cfg["media_type"].(string)
			}
			repoURLs = getNormalizedImageRepoURLs(
				// GitHub sometimes sends package URLs with a trailing colon and no tag
				// (e.g., "ghcr.io/user/image:"). Such strings are not valid OCI image
				// references. We trim the trailing colon to avoid parsing errors.
				//
				// TODO(krancour): We do not have a firm grasp on why this sometimes
				// happens and sometimes does not.
				strings.TrimSuffix(pkgVer.GetPackageURL(), ":"),
				mediaType,
			)
			tag := containerMeta.GetTag().GetName()
			qualifiers = []string{tag}
			logger = logger.WithValues(
				"mediaType", mediaType,
				"tag", tag,
			)
		}

		logger = logger.WithValues("repoURLs", repoURLs)
		ctx = logging.ContextWithLogger(ctx, logger)

		refreshWarehouses(ctx, w, g.client, g.project, repoURLs, qualifiers...)
	})
}
