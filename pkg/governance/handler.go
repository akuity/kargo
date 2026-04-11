package governance

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/goccy/go-yaml"
	"github.com/google/go-github/v76/github"

	"github.com/akuity/kargo/pkg/logging"
)

const (
	maxBodySize = 1 << 20 // 1 MB
	configPath  = ".github/governance.yaml"
)

// handler is an http.Handler that processes inbound GitHub webhook requests.
type handler struct {
	webhookSecret []byte
	clientFactory GitHubClientFactory
}

// NewHandler returns an http.Handler that processes inbound GitHub webhook
// requests. The webhookSecret is used to validate webhook signatures. The
// clientFactory is used to create authenticated GitHub clients per
// installation.
func NewHandler(
	webhookSecret []byte,
	clientFactory GitHubClientFactory,
) http.Handler {
	return &handler{
		webhookSecret: webhookSecret,
		clientFactory: clientFactory,
	}
}

// ServeHTTP implements http.Handler.
func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := logging.LoggerFromContext(r.Context())

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		logger.Error(err, "error reading request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sig := r.Header.Get(github.SHA256SignatureHeader)
	if sig == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if err = github.ValidateSignature(sig, body, h.webhookSecret); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	eventType := r.Header.Get(github.EventTypeHeader)
	logger = logger.WithValues("eventType", eventType)

	event, err := github.ParseWebHook(eventType, body)
	if err != nil {
		logger.Error(err, "error parsing webhook payload")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ctx := logging.ContextWithLogger(r.Context(), logger)

	// Narrow to the events we actually handle and extract common repo info.
	var (
		owner, repo    string
		installationID int64
	)
	switch e := event.(type) {
	case *github.IssueCommentEvent:
		if e.GetAction() != "created" {
			logger.Debug("ignoring non-created issue comment event", "action", e.GetAction())
			w.WriteHeader(http.StatusNoContent)
			return
		}
		owner, repo, installationID = h.repoInfo(e)
	case *github.IssuesEvent:
		if e.GetAction() != "opened" {
			logger.Debug("ignoring non-opened issues event", "action", e.GetAction())
			w.WriteHeader(http.StatusNoContent)
			return
		}
		owner, repo, installationID = h.repoInfo(e)
	case *github.PullRequestEvent:
		if e.GetAction() != "opened" {
			logger.Debug("ignoring non-opened pull request event", "action", e.GetAction())
			w.WriteHeader(http.StatusNoContent)
			return
		}
		owner, repo, installationID = h.repoInfo(e)
	case *github.PingEvent:
		logger.Debug("received ping event")
		w.WriteHeader(http.StatusOK)
		return
	default:
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Load config once up front for any event we're going to handle.
	reposClient, err := h.clientFactory.NewRepositoriesClient(installationID)
	if err != nil {
		logger.Error(err, "error creating repositories client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	cfg, err := h.loadConfig(ctx, reposClient, owner, repo)
	if err != nil {
		logger.Error(err, "error loading config")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	issuesClient, err := h.clientFactory.NewIssuesClient(installationID)
	if err != nil {
		logger.Error(err, "error creating issues client")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	logger = logger.WithValues("owner", owner, "repo", repo)
	ctx = logging.ContextWithLogger(ctx, logger)

	switch e := event.(type) {
	case *github.IssueCommentEvent:
		var prsClient PullRequestsClient
		prsClient, err = h.clientFactory.NewPullRequestsClient(installationID)
		if err != nil {
			logger.Error(err, "error creating pull requests client")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		commentHandler := &commentHandler{
			cfg:          cfg,
			owner:        owner,
			repo:         repo,
			issuesClient: issuesClient,
			prsClient:    prsClient,
		}
		err = commentHandler.handleCreated(ctx, e)
	case *github.IssuesEvent:
		if cfg.Issues != nil {
			issueHandler := &issueHandler{
				cfg:          *cfg.Issues,
				owner:        owner,
				repo:         repo,
				issuesClient: issuesClient,
			}
			err = issueHandler.handleOpened(ctx, e)
		}
	case *github.PullRequestEvent:
		var prsClient PullRequestsClient
		prsClient, err = h.clientFactory.NewPullRequestsClient(installationID)
		if err != nil {
			logger.Error(err, "error creating pull requests client")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		prHandler := &prHandler{
			cfg:          cfg,
			owner:        owner,
			repo:         repo,
			issuesClient: issuesClient,
			prsClient:    prsClient,
		}
		err = prHandler.handleOpened(ctx, e)
	}
	if err != nil {
		logger.Error(err, "error handling event")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// repoInfo extracts the owner, repo, and installation ID from any go-github
// event that carries a Repo and an Installation (issue comment, issues, pull
// request, etc.).
func (h *handler) repoInfo(
	e interface {
		GetRepo() *github.Repository
		GetInstallation() *github.Installation
	},
) (owner, repo string, installationID int64) {
	return e.GetRepo().GetOwner().GetLogin(),
		e.GetRepo().GetName(),
		e.GetInstallation().GetID()
}

// loadConfig loads the governance config for the given repository. It returns
// an error if the config cannot be loaded or parsed. Note that the config is
// not cached, so it will be fetched and parsed on every event.
func (h *handler) loadConfig(
	ctx context.Context,
	reposClient RepositoriesClient,
	owner string,
	repo string,
) (config, error) {
	content, _, _, err := reposClient.GetContents(
		ctx, owner, repo, configPath,
		&github.RepositoryContentGetOptions{Ref: "HEAD"},
	)
	if err != nil {
		return config{}, fmt.Errorf("error fetching governance config: %w", err)
	}
	if content == nil {
		return config{}, fmt.Errorf("governance config not found at %s", configPath)
	}
	raw, err := content.GetContent()
	if err != nil {
		return config{}, fmt.Errorf("error decoding governance config: %w", err)
	}
	cfg := config{}
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		return config{}, fmt.Errorf("error parsing governance config: %w", err)
	}
	return cfg, nil
}
