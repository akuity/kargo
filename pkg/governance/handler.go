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
		action := e.GetAction()
		if action != "opened" &&
			action != "reopened" &&
			action != "ready_for_review" {
			logger.Debug("ignoring pull request event", "action", action)
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
	if cfg == nil {
		logger.Debug("no config found, skipping event")
		w.WriteHeader(http.StatusNoContent)
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
			cfg:          *cfg,
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
			cfg:          *cfg,
			owner:        owner,
			repo:         repo,
			issuesClient: issuesClient,
			prsClient:    prsClient,
		}
		opts := &handlePROpenedOpts{}
		if e.GetAction() == "reopened" || e.GetAction() == "ready_for_review" {
			opts.applyPolicyOnly = true
		}
		err = prHandler.handleOpened(ctx, e, opts)
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

// loadConfig loads and returns the governance config for the given repository.
// It returns nil and no error if no such config exists. It returns nil and an
// error if the config cannot be loaded or parsed for any other reason. Note
// that the config is not cached, so it will be fetched and parsed on every
// event.
func (h *handler) loadConfig(
	ctx context.Context,
	reposClient RepositoriesClient,
	owner string,
	repo string,
) (*config, error) {
	content, _, resp, err := reposClient.GetContents(
		ctx, owner, repo, configPath,
		&github.RepositoryContentGetOptions{Ref: "HEAD"},
	)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("error fetching governance config: %w", err)
	}
	raw, err := content.GetContent()
	if err != nil {
		return nil, fmt.Errorf("error decoding governance config: %w", err)
	}
	cfg := config{}
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil, fmt.Errorf("error parsing governance config: %w", err)
	}
	return &cfg, nil
}
