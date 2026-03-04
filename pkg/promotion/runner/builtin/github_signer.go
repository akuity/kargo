package builtin

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/go-github/v76/github"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/kelseyhightower/envconfig"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/urls"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const (
	stepKindGitHubSign = "github-sign"

	// compareStatusAhead is the GitHub Compare API status indicating that the
	// base is behind the head (i.e. there are revisions to replay).
	compareStatusAhead     = "ahead"
	compareStatusIdentical = "identical"
)

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindGitHubSign,
			Metadata: promotion.StepRunnerMetadata{
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessCredentials,
				},
			},
			Value: newGitHubSigner,
		},
	)
}

// githubSignClient is an interface for the GitHub API methods used by the
// github-sign step. This enables unit testing with mocks.
type githubSignClient interface {
	CompareCommits(
		ctx context.Context,
		owner, repo, base, head string,
		opts *github.ListOptions,
	) (*github.CommitsComparison, *github.Response, error)
	CreateCommit(
		ctx context.Context,
		owner, repo string,
		commit github.Commit,
		opts *github.CreateCommitOptions,
	) (*github.Commit, *github.Response, error)
	UpdateRef(
		ctx context.Context,
		owner, repo, ref string,
		updateRef github.UpdateRef,
	) (*github.Reference, *github.Response, error)
}

// githubSignClientWrapper wraps a *github.Client to implement
// githubSignClient.
type githubSignClientWrapper struct {
	client *github.Client
}

func (w *githubSignClientWrapper) CompareCommits(
	ctx context.Context,
	owner, repo, base, head string,
	opts *github.ListOptions,
) (*github.CommitsComparison, *github.Response, error) {
	return w.client.Repositories.CompareCommits(ctx, owner, repo, base, head, opts)
}

func (w *githubSignClientWrapper) CreateCommit(
	ctx context.Context,
	owner, repo string,
	commit github.Commit,
	opts *github.CreateCommitOptions,
) (*github.Commit, *github.Response, error) {
	return w.client.Git.CreateCommit(ctx, owner, repo, commit, opts)
}

func (w *githubSignClientWrapper) UpdateRef(
	ctx context.Context,
	owner, repo, ref string,
	updateRef github.UpdateRef,
) (*github.Reference, *github.Response, error) {
	return w.client.Git.UpdateRef(ctx, owner, repo, ref, updateRef)
}

// gitHubSignerConfig holds controller-level configuration for the github-sign
// step, populated from environment variables.
type gitHubSignerConfig struct {
	MaxRevisions int `envconfig:"GITHUB_SIGN_MAX_REVISIONS" default:"10"`
}

// gitHubSigner is an implementation of the promotion.StepRunner interface that
// signs revisions on a GitHub branch by replaying them via the GitHub REST API.
type gitHubSigner struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
	cfg          gitHubSignerConfig
}

// newGitHubSigner returns an implementation of the promotion.StepRunner
// interface that signs revisions on a GitHub branch.
func newGitHubSigner(caps promotion.StepRunnerCapabilities) promotion.StepRunner {
	cfg := gitHubSignerConfig{}
	envconfig.MustProcess("", &cfg)
	return &gitHubSigner{
		credsDB:      caps.CredsDB,
		schemaLoader: getConfigSchemaLoader(stepKindGitHubSign),
		cfg:          cfg,
	}
}

// Run implements the promotion.StepRunner interface.
func (g *gitHubSigner) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	cfg, err := g.convert(stepCtx.Config)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, &promotion.TerminalError{Err: err}
	}
	return g.run(ctx, stepCtx, cfg)
}

// convert validates the configuration against a JSON schema and converts it
// into a builtin.GitHubSignConfig struct.
func (g *gitHubSigner) convert(cfg promotion.Config) (builtin.GitHubSignConfig, error) {
	return validateAndConvert[builtin.GitHubSignConfig](
		g.schemaLoader, cfg, stepKindGitHubSign,
	)
}

func (g *gitHubSigner) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.GitHubSignConfig,
) (promotion.StepResult, error) {
	logger := logging.LoggerFromContext(ctx)

	// Short-circuit if shared state has output from a previous execution of
	// this step that contains a commit SHA matching the current branch HEAD.
	if prevOutput, err := g.getPreviousOutput(stepCtx); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error checking previous step output: %w", err)
	} else if prevOutput != nil {
		logger.Debug(
			"reusing output from previous execution",
			"commit", prevOutput["commit"],
		)
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusSucceeded,
			Output: prevOutput,
		}, nil
	}

	logger.Debug(
		"signing revision range",
		"repoURL", cfg.RepoURL,
		"targetBranch", cfg.TargetBranch,
		"head", cfg.Head,
		"base", cfg.Base,
	)

	// Get credentials for the repository.
	creds, err := g.credsDB.Get(
		ctx,
		stepCtx.Project,
		credentials.TypeGit,
		cfg.RepoURL,
	)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error getting credentials for %s: %w", cfg.RepoURL, err)
	}

	var token string
	if creds != nil {
		token = creds.Password
	}
	if token == "" {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{
				Err: fmt.Errorf(
					"no credentials (GitHub App installation token) found for %s",
					cfg.RepoURL,
				),
			}
	}

	// Parse repo URL and create GitHub client.
	owner, repo, ghClient, err := g.newGitHubClient(
		cfg.RepoURL,
		token,
		cfg.InsecureSkipTLSVerify,
	)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error creating GitHub client: %w", err)
	}

	// Sign the revision range.
	result, err := g.signRevisionRange(ctx, ghClient, owner, repo, cfg)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}

	return result, nil
}

// signRevisionRange enumerates revisions in the range base..head and replays
// them as signed revisions via the GitHub REST API.
func (g *gitHubSigner) signRevisionRange(
	ctx context.Context,
	client githubSignClient,
	owner, repo string,
	cfg builtin.GitHubSignConfig,
) (promotion.StepResult, error) {
	logger := logging.LoggerFromContext(ctx)
	maxRevisions := g.cfg.MaxRevisions

	// Enumerate revisions in the range base..head.
	comparison, _, err := client.CompareCommits(
		ctx, owner, repo, cfg.Base, cfg.Head, nil,
	)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf(
				"error comparing %s...%s: %w",
				cfg.Base, cfg.Head, err,
			)
	}

	status := comparison.GetStatus()
	logger.Debug(
		"compared revision range",
		"base", cfg.Base,
		"head", cfg.Head,
		"status", status,
		"aheadBy", comparison.GetAheadBy(),
		"totalCommits", comparison.GetTotalCommits(),
	)
	switch status {
	case compareStatusAhead:
		// Expected: base is behind head.
	case compareStatusIdentical:
		// No revisions to sign.
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusSkipped,
			Message: fmt.Sprintf(
				"no revisions to sign: %s and %s are identical",
				cfg.Base, cfg.Head,
			),
			Output: map[string]any{
				"commit": cfg.Base,
				"branch": cfg.TargetBranch,
			},
		}, nil
	default:
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{
				Err: fmt.Errorf(
					"cannot sign revision range %s..%s: comparison status is %q "+
						"(base must be an ancestor of head)",
					cfg.Base, cfg.Head, status,
				),
			}
	}

	commits := comparison.Commits
	if len(commits) == 0 {
		return promotion.StepResult{
			Status:  kargoapi.PromotionStepStatusSkipped,
			Message: "no revisions to sign in range",
			Output: map[string]any{
				"commit": cfg.Base,
				"branch": cfg.TargetBranch,
			},
		}, nil
	}

	if len(commits) > maxRevisions {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{
				Err: fmt.Errorf(
					"revision range %s..%s contains %d revisions, "+
						"which exceeds the maximum of %d "+
						"(configurable via GITHUB_SIGN_MAX_REVISIONS env var)",
					cfg.Base, cfg.Head, len(commits), maxRevisions,
				),
			}
	}

	// Replay each revision as a signed commit via the API.
	parentSHA := cfg.Base
	var lastSignedSHA string
	for i, rc := range commits {
		if rc.Commit == nil || rc.Commit.Tree == nil || rc.Commit.Tree.SHA == nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("revision %d in range has missing tree information", i)
		}
		message := rc.Commit.GetMessage()

		newCommit, _, createErr := client.CreateCommit(
			ctx, owner, repo,
			github.Commit{
				Message: &message,
				Tree:    &github.Tree{SHA: rc.Commit.Tree.SHA},
				Parents: []*github.Commit{{SHA: &parentSHA}},
			},
			nil, // No create options — let GitHub set author/committer and sign.
		)
		if createErr != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf(
					"error creating signed revision %d/%d (original: %s): %w",
					i+1, len(commits), rc.GetSHA(), createErr,
				)
		}
		lastSignedSHA = newCommit.GetSHA()
		parentSHA = lastSignedSHA
		logger.Debug(
			"created signed revision",
			"index", i+1,
			"total", len(commits),
			"original", rc.GetSHA(),
			"signed", lastSignedSHA,
		)
	}

	// Update the target branch ref to point to the final signed revision.
	branchRef := "heads/" + cfg.TargetBranch
	_, _, err = client.UpdateRef(
		ctx, owner, repo,
		branchRef,
		github.UpdateRef{
			SHA:   lastSignedSHA,
			Force: new(bool),
		},
	)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error updating ref %s to %s: %w", branchRef, lastSignedSHA, err)
	}
	logger.Debug("updated branch ref", "ref", branchRef, "sha", lastSignedSHA)

	commitURL := g.buildCommitURL(cfg.RepoURL, lastSignedSHA)

	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: map[string]any{
			"commit":    lastSignedSHA,
			"commitURL": commitURL,
			"branch":    cfg.TargetBranch,
		},
	}, nil
}

// getPreviousOutput checks shared state for output from a previous execution
// of this step. If found, returns the output map; otherwise returns nil.
func (g *gitHubSigner) getPreviousOutput(
	stepCtx *promotion.StepContext,
) (map[string]any, error) {
	stepOutput, exists := stepCtx.SharedState.Get(stepCtx.Alias)
	if !exists {
		return nil, nil
	}
	outputMap, ok := stepOutput.(map[string]any)
	if !ok {
		return nil, fmt.Errorf(
			"output from step with alias %q is not a map[string]any",
			stepCtx.Alias,
		)
	}
	if _, hasCommit := outputMap["commit"]; !hasCommit {
		return nil, nil
	}
	return outputMap, nil
}

// newGitHubClient parses a repository URL and creates an authenticated GitHub
// API client. Returns the owner, repo name, and client.
func (g *gitHubSigner) newGitHubClient(
	repoURL, token string,
	insecureSkipTLSVerify bool,
) (string, string, githubSignClient, error) {
	u, err := url.Parse(urls.NormalizeGit(repoURL))
	if err != nil {
		return "", "", nil, fmt.Errorf(
			"error parsing repository URL %q: %w", repoURL, err,
		)
	}

	scheme := u.Scheme
	if scheme != "https" && scheme != "http" {
		scheme = "https"
	}

	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", "", nil, fmt.Errorf(
			"could not extract repository owner and name from URL %q", repoURL,
		)
	}
	owner, repo := parts[0], parts[1]

	httpClient := cleanhttp.DefaultClient()
	if insecureSkipTLSVerify {
		transport := cleanhttp.DefaultTransport()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
		httpClient.Transport = transport
	}

	client := github.NewClient(httpClient)
	if u.Host != "github.com" {
		baseURL := fmt.Sprintf("%s://%s", scheme, u.Host)
		client, err = client.WithEnterpriseURLs(baseURL, baseURL)
		if err != nil {
			return "", "", nil, fmt.Errorf(
				"error configuring GitHub Enterprise URLs: %w", err,
			)
		}
	}
	client = client.WithAuthToken(token)

	return owner, repo, &githubSignClientWrapper{client: client}, nil
}

// buildCommitURL constructs a human-readable commit URL from a repository URL
// and commit SHA.
func (g *gitHubSigner) buildCommitURL(repoURL, sha string) string {
	normalizedURL := urls.NormalizeGit(repoURL)
	parsedURL, err := url.Parse(normalizedURL)
	if err != nil {
		return ""
	}
	return fmt.Sprintf(
		"https://%s%s/commit/%s", parsedURL.Host, parsedURL.Path, sha,
	)
}
