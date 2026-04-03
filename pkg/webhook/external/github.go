package external

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
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
	githubEventTypePackage     = "package"
	githubEventTypePing        = "ping"
	githubEventTypePullRequest = "pull_request"
	githubEventTypePush        = "push"
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
			githubEventTypePullRequest,
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

		switch e := event.(type) {
		case *gh.PackageEvent:
			g.handlePackageEvent(ctx, w, e)
		case *gh.PingEvent:
			g.handlePingEvent(w)
		case *gh.PullRequestEvent:
			g.handlePullRequestEvent(ctx, w, e)
		case *gh.PushEvent:
			g.handlePushEvent(ctx, w, e)
		case *gh.RegistryPackageEvent:
			g.handleRegistryPackageEvent(ctx, w, e)
		}
	})
}

func (g *githubWebhookReceiver) handlePackageEvent(
	ctx context.Context,
	w http.ResponseWriter,
	e *gh.PackageEvent,
) {
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
	repoURLs := getNormalizedImageRepoURLs(
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

	logger := logging.LoggerFromContext(ctx)
	logger = logger.WithValues(
		"mediaType", mediaType,
		"tag", tag,
		"repoURLs", repoURLs,
	)
	ctx = logging.ContextWithLogger(ctx, logger)
	refreshWarehouses(ctx, w, g.client, g.project, repoURLs, nil, tag)
}

func (g *githubWebhookReceiver) handlePingEvent(w http.ResponseWriter) {
	xhttp.WriteResponseJSON(
		w,
		http.StatusOK,
		map[string]string{
			"msg": "ping event received, webhook is configured correctly",
		},
	)
}

func (g *githubWebhookReceiver) handlePullRequestEvent(
	ctx context.Context,
	w http.ResponseWriter,
	e *gh.PullRequestEvent,
) {
	if e.GetAction() != "closed" {
		xhttp.WriteResponseJSON(w, http.StatusOK, nil)
		return
	}
	prURL := e.GetPullRequest().GetHTMLURL()
	logger := logging.LoggerFromContext(ctx)
	logger = logger.WithValues("prURL", prURL)
	ctx = logging.ContextWithLogger(ctx, logger)
	refreshPromotionsByPrURL(ctx, w, g.client, g.project, prURL)
}

func (g *githubWebhookReceiver) handlePushEvent(
	ctx context.Context,
	w http.ResponseWriter,
	e *gh.PushEvent,
) {
	repoURLs := []string{
		urls.NormalizeGit(e.GetRepo().GetCloneURL()),
		urls.NormalizeGit(e.GetRepo().GetSSHURL()),
	}
	ref := e.GetRef()

	// GitHub includes a max of 2048 commits in the push event payload. If
	// there are more than that, you need to use the GitHub API to retrieve
	// the full list of commits. We're not going to do that. Instead, for such
	// a case, we'll let the list of changed files remain empty, the effect of
	// which will be that all Warehouses subscribed to the repository will be
	// refreshed instead of just the ones whose filters match the changed
	// files. This is good enough for this edge case. So we proceed only if
	// there are fewer than 2048 commits in the payload.
	//
	// Also note: e.Commits is deprecated with a note that GitHub will be
	// removing commit details from event payloads beginning October 7, 2025,
	// however, as of this writing (March 18, 2026) GitHub is still including
	// commit details in the payloads. GitHub's own API documentation for the
	// push event does not indicate these are deprecated.
	//
	// See: https://docs.github.com/en/webhooks/webhook-events-and-payloads#push
	var changedFiles []string
	if len(e.Commits) < 2048 { // nolint: staticcheck
		commits := e.Commits // nolint: staticcheck
		if e.HeadCommit != nil {
			commits = append(commits, e.HeadCommit)
		}
		changedFiles = collectPaths(commits, func(c *gh.HeadCommit) []string {
			return slices.Concat(c.Added, c.Modified, c.Removed)
		})
	}

	logger := logging.LoggerFromContext(ctx)
	logger = logger.WithValues(
		"repoURLs", repoURLs,
		"ref", ref,
	)
	ctx = logging.ContextWithLogger(ctx, logger)
	refreshWarehouses(ctx, w, g.client, g.project, repoURLs, changedFiles, ref)
}

func (g *githubWebhookReceiver) handleRegistryPackageEvent(
	ctx context.Context,
	w http.ResponseWriter,
	e *gh.RegistryPackageEvent,
) {
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
	repoURLs := getNormalizedImageRepoURLs(
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

	logger := logging.LoggerFromContext(ctx)
	logger = logger.WithValues(
		"mediaType", mediaType,
		"tag", tag,
		"repoURLs", repoURLs,
	)
	ctx = logging.ContextWithLogger(ctx, logger)
	refreshWarehouses(ctx, w, g.client, g.project, repoURLs, nil, tag)
}
