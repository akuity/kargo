package builtin

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"
	"sync"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/google/go-github/v76/github"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/kelseyhightower/envconfig"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/urls"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const (
	stepKindGitHubVerifiedPush = "github-verified-push"

	// stagingRefPrefix is the ref namespace used for staging refs. These are
	// not under refs/heads/ so they do not appear as branches in GitHub's UI.
	stagingRefPrefix = "refs/kargo/staging"

	// compareStatusAhead is the GitHub Compare API status indicating that the
	// base is behind the head (i.e. there are revisions to replay).
	compareStatusAhead     = "ahead"
	compareStatusIdentical = "identical"
)

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindGitHubVerifiedPush,
			Metadata: promotion.StepRunnerMetadata{
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessCredentials,
				},
			},
			Value: newGitHubVerifiedPusher,
		},
	)
}

// githubVerifiedPushClient is an interface for the GitHub API methods used by
// the github-verified-push step. This enables unit testing with mocks.
type githubVerifiedPushClient interface {
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
	GetRef(
		ctx context.Context,
		owner, repo, ref string,
	) (*github.Reference, *github.Response, error)
	UpdateRef(
		ctx context.Context,
		owner, repo, ref string,
		updateRef github.UpdateRef,
	) (*github.Reference, *github.Response, error)
	DeleteRef(
		ctx context.Context,
		owner, repo, ref string,
	) (*github.Response, error)
}

// githubVerifiedPushClientWrapper wraps a *github.Client to implement
// githubVerifiedPushClient.
type githubVerifiedPushClientWrapper struct {
	client *github.Client
}

func (w *githubVerifiedPushClientWrapper) CompareCommits(
	ctx context.Context,
	owner, repo, base, head string,
	opts *github.ListOptions,
) (*github.CommitsComparison, *github.Response, error) {
	return w.client.Repositories.CompareCommits(
		ctx, owner, repo, base, head, opts,
	)
}

func (w *githubVerifiedPushClientWrapper) CreateCommit(
	ctx context.Context,
	owner, repo string,
	commit github.Commit,
	opts *github.CreateCommitOptions,
) (*github.Commit, *github.Response, error) {
	return w.client.Git.CreateCommit(ctx, owner, repo, commit, opts)
}

func (w *githubVerifiedPushClientWrapper) GetRef(
	ctx context.Context,
	owner, repo, ref string,
) (*github.Reference, *github.Response, error) {
	return w.client.Git.GetRef(ctx, owner, repo, ref)
}

func (w *githubVerifiedPushClientWrapper) UpdateRef(
	ctx context.Context,
	owner, repo, ref string,
	updateRef github.UpdateRef,
) (*github.Reference, *github.Response, error) {
	return w.client.Git.UpdateRef(ctx, owner, repo, ref, updateRef)
}

func (w *githubVerifiedPushClientWrapper) DeleteRef(
	ctx context.Context,
	owner, repo, ref string,
) (*github.Response, error) {
	return w.client.Git.DeleteRef(ctx, owner, repo, ref)
}

// githubVerifiedPusherConfig holds controller-level configuration for the
// github-verified-push step, populated from environment variables.
type githubVerifiedPusherConfig struct {
	MaxRevisions int `envconfig:"GITHUB_VERIFIED_PUSH_MAX_REVISIONS" default:"10"`
}

// githubVerifiedPusher is an implementation of the promotion.StepRunner
// interface that pushes local commits to a GitHub repository as verified
// (signed) commits using the GitHub REST API.
type githubVerifiedPusher struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
	cfg          githubVerifiedPusherConfig
	branchMus    map[string]*sync.Mutex
	masterMu     sync.Mutex
}

// newGitHubVerifiedPusher returns an implementation of the
// promotion.StepRunner interface that pushes local commits to a GitHub
// repository as verified commits.
func newGitHubVerifiedPusher(
	caps promotion.StepRunnerCapabilities,
) promotion.StepRunner {
	cfg := githubVerifiedPusherConfig{}
	envconfig.MustProcess("", &cfg)
	return &githubVerifiedPusher{
		credsDB:      caps.CredsDB,
		schemaLoader: getConfigSchemaLoader(stepKindGitHubVerifiedPush),
		cfg:          cfg,
		branchMus:    map[string]*sync.Mutex{},
	}
}

// Run implements the promotion.StepRunner interface.
func (g *githubVerifiedPusher) Run(
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
// into a builtin.GitHubVerifiedPushConfig struct.
func (g *githubVerifiedPusher) convert(
	cfg promotion.Config,
) (builtin.GitHubVerifiedPushConfig, error) {
	return validateAndConvert[builtin.GitHubVerifiedPushConfig](
		g.schemaLoader, cfg, stepKindGitHubVerifiedPush,
	)
}

func (g *githubVerifiedPusher) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.GitHubVerifiedPushConfig,
) (promotion.StepResult, error) {
	logger := logging.LoggerFromContext(ctx)

	// Short-circuit if shared state has output from a previous execution of
	// this step that contains a commit SHA.
	if prevOutput, err := g.getPreviousOutput(stepCtx); err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusErrored,
		}, fmt.Errorf("error checking previous step output: %w", err)
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

	// Load the working tree to get the repository URL.
	path, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusErrored,
			}, fmt.Errorf(
				"error joining path %s with work dir %s: %w",
				cfg.Path, stepCtx.WorkDir, err,
			)
	}
	loadOpts := &git.LoadWorkTreeOptions{}
	workTree, err := git.LoadWorkTree(path, loadOpts)
	if err != nil {
		return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusErrored,
			}, fmt.Errorf(
				"error loading working tree from %s: %w", cfg.Path, err,
			)
	}

	// Fetch credentials for the repository.
	creds, err := g.credsDB.Get(
		ctx, stepCtx.Project, credentials.TypeGit, workTree.URL(),
	)
	if err != nil {
		return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusErrored,
			}, fmt.Errorf(
				"error getting credentials for %s: %w", workTree.URL(), err,
			)
	}
	if creds != nil {
		loadOpts.Credentials = &git.RepoCredentials{
			Username:      creds.Username,
			Password:      creds.Password,
			SSHPrivateKey: creds.SSHPrivateKey,
		}
	}
	if workTree, err = git.LoadWorkTree(path, loadOpts); err != nil {
		return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusErrored,
			}, fmt.Errorf(
				"error loading working tree from %s: %w", cfg.Path, err,
			)
	}

	var token string
	if creds != nil {
		token = creds.Password
	}
	if token == "" {
		return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusFailed,
			}, &promotion.TerminalError{
				Err: fmt.Errorf(
					"no credentials (GitHub App installation token) found for %s",
					workTree.URL(),
				),
			}
	}

	// Resolve the target branch.
	targetBranch := cfg.TargetBranch
	if cfg.GenerateTargetBranch {
		targetBranch = fmt.Sprintf(
			"kargo/promotion/%s", stepCtx.Promotion,
		)
	}
	if targetBranch == "" {
		if targetBranch, err = workTree.CurrentBranch(); err != nil {
			return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusErrored,
			}, fmt.Errorf("error getting current branch: %w", err)
		}
	}

	// Get the local HEAD SHA — this is what we'll push to the staging ref.
	localHead, err := workTree.LastCommitID()
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusErrored,
		}, fmt.Errorf("error getting local HEAD: %w", err)
	}

	// Push local commits to a non-branch staging ref. This gets all objects
	// onto GitHub without creating a visible branch.
	stagingRef := fmt.Sprintf(
		"%s/%s", stagingRefPrefix, stepCtx.Promotion,
	)
	logger.Debug(
		"pushing to staging ref",
		"ref", stagingRef,
		"localHead", localHead,
	)

	g.acquireBranchLock(workTree.URL(), targetBranch)
	defer g.releaseBranchLock(workTree.URL(), targetBranch)

	if err = workTree.Push(&git.PushOptions{
		TargetBranch: stagingRef,
		Force:        true,
	}); err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusErrored,
		}, fmt.Errorf("error pushing to staging ref %s: %w", stagingRef, err)
	}

	// Create the GitHub client.
	owner, repo, ghClient, err := g.newGitHubClient(
		workTree.URL(), token, cfg.InsecureSkipTLSVerify,
	)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusErrored,
		}, fmt.Errorf("error creating GitHub client: %w", err)
	}

	// Ensure the staging ref is cleaned up regardless of outcome.
	defer g.cleanupStagingRef(ctx, ghClient, owner, repo, stagingRef)

	// Get the current target branch HEAD from GitHub.
	targetRef := "heads/" + targetBranch
	targetHeadRef, _, err := ghClient.GetRef(ctx, owner, repo, targetRef)
	if err != nil {
		return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusErrored,
			}, fmt.Errorf(
				"error getting ref %s: %w", targetRef, err,
			)
	}
	targetHead := targetHeadRef.GetObject().GetSHA()

	logger.Debug(
		"signing revision range",
		"targetBranch", targetBranch,
		"targetHead", targetHead,
		"localHead", localHead,
	)

	// Enumerate commits to sign using the Compare API.
	result, err := g.signAndUpdate(
		ctx, ghClient, owner, repo, targetBranch, targetHead, localHead,
		workTree.URL(),
	)
	if err != nil {
		return result, err
	}

	return result, nil
}

// signAndUpdate enumerates commits in the range targetHead..localHead,
// replays them as signed commits via the GitHub REST API, and updates the
// target branch ref to point to the final signed commit.
func (g *githubVerifiedPusher) signAndUpdate(
	ctx context.Context,
	client githubVerifiedPushClient,
	owner, repo, targetBranch, targetHead, localHead, repoURL string,
) (promotion.StepResult, error) {
	logger := logging.LoggerFromContext(ctx)

	comparison, _, err := client.CompareCommits(
		ctx, owner, repo, targetHead, localHead, nil,
	)
	if err != nil {
		return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusErrored,
			}, fmt.Errorf(
				"error comparing %s...%s: %w", targetHead, localHead, err,
			)
	}

	status := comparison.GetStatus()
	logger.Debug(
		"compared revision range",
		"targetHead", targetHead,
		"localHead", localHead,
		"status", status,
		"aheadBy", comparison.GetAheadBy(),
		"totalCommits", comparison.GetTotalCommits(),
	)

	switch status {
	case compareStatusAhead:
		// Expected: target is behind local head.
	case compareStatusIdentical:
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusSkipped,
			Message: fmt.Sprintf(
				"no revisions to sign: %s and %s are identical",
				targetHead, localHead,
			),
			Output: map[string]any{
				stateKeyCommit: targetHead,
				stateKeyBranch: targetBranch,
			},
		}, nil
	default:
		return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusFailed,
			}, &promotion.TerminalError{
				Err: fmt.Errorf(
					"cannot sign revision range %s..%s: "+
						"comparison status is %q "+
						"(target branch may have diverged)",
					targetHead, localHead, status,
				),
			}
	}

	commits := comparison.Commits
	if len(commits) == 0 {
		return promotion.StepResult{
			Status:  kargoapi.PromotionStepStatusSkipped,
			Message: "no revisions to sign in range",
			Output: map[string]any{
				stateKeyCommit: targetHead,
				stateKeyBranch: targetBranch,
			},
		}, nil
	}

	if len(commits) > g.cfg.MaxRevisions {
		return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusFailed,
			}, &promotion.TerminalError{
				Err: fmt.Errorf(
					"revision range %s..%s contains %d revisions, "+
						"which exceeds the maximum of %d "+
						"(configurable via "+
						"GITHUB_VERIFIED_PUSH_MAX_REVISIONS env var)",
					targetHead, localHead, len(commits), g.cfg.MaxRevisions,
				),
			}
	}

	// Replay each revision as a signed commit via the API.
	parentSHA := targetHead
	var lastSignedSHA string
	for i, rc := range commits {
		if rc.Commit == nil ||
			rc.Commit.Tree == nil ||
			rc.Commit.Tree.SHA == nil {
			return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, fmt.Errorf(
					"revision %d in range has missing tree information", i,
				)
		}
		message := rc.Commit.GetMessage()

		newCommit, _, createErr := client.CreateCommit(
			ctx, owner, repo,
			github.Commit{
				Message: &message,
				Tree:    &github.Tree{SHA: rc.Commit.Tree.SHA},
				Parents: []*github.Commit{{SHA: &parentSHA}},
			},
			nil, // Let GitHub set author/committer and sign.
		)
		if createErr != nil {
			return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, fmt.Errorf(
					"error creating signed revision %d/%d "+
						"(original: %s): %w",
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

	// Update the target branch ref to the final signed commit. Using
	// force=false ensures this is a fast-forward update only.
	targetRef := "heads/" + targetBranch
	_, _, err = client.UpdateRef(
		ctx, owner, repo, targetRef,
		github.UpdateRef{
			SHA:   lastSignedSHA,
			Force: new(bool),
		},
	)
	if err != nil {
		return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusErrored,
			}, fmt.Errorf(
				"error updating ref %s to %s: %w",
				targetRef, lastSignedSHA, err,
			)
	}
	logger.Debug(
		"updated branch ref",
		"ref", targetRef,
		"sha", lastSignedSHA,
	)

	commitURL := g.buildCommitURL(repoURL, lastSignedSHA)

	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: map[string]any{
			stateKeyCommit:    lastSignedSHA,
			stateKeyCommitURL: commitURL,
			stateKeyBranch:    targetBranch,
		},
	}, nil
}

// cleanupStagingRef deletes the staging ref from GitHub. Errors are logged
// but do not cause the step to fail, since the target branch has already been
// updated at this point.
func (g *githubVerifiedPusher) cleanupStagingRef(
	ctx context.Context,
	client githubVerifiedPushClient,
	owner, repo, stagingRef string,
) {
	logger := logging.LoggerFromContext(ctx)
	// The API expects the ref without the "refs/" prefix.
	ref := strings.TrimPrefix(stagingRef, "refs/")
	if _, err := client.DeleteRef(ctx, owner, repo, ref); err != nil {
		logger.Error(
			err,
			"error deleting staging ref (non-fatal)",
			"ref", stagingRef,
		)
	} else {
		logger.Debug("deleted staging ref", "ref", stagingRef)
	}
}

// getPreviousOutput checks shared state for output from a previous execution
// of this step. If found, returns the output map; otherwise returns nil.
func (g *githubVerifiedPusher) getPreviousOutput(
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

// acquireBranchLock obtains a per-branch mutex to serialize concurrent
// operations targeting the same branch.
func (g *githubVerifiedPusher) acquireBranchLock(repoURL, branch string) {
	key := fmt.Sprintf("%s:%s", repoURL, branch)
	g.masterMu.Lock()
	if _, exists := g.branchMus[key]; !exists {
		g.branchMus[key] = &sync.Mutex{}
	}
	mu := g.branchMus[key]
	g.masterMu.Unlock()
	mu.Lock()
}

// releaseBranchLock releases the per-branch mutex.
func (g *githubVerifiedPusher) releaseBranchLock(repoURL, branch string) {
	key := fmt.Sprintf("%s:%s", repoURL, branch)
	g.masterMu.Lock()
	mu := g.branchMus[key]
	g.masterMu.Unlock()
	mu.Unlock()
}

// newGitHubClient parses a repository URL and creates an authenticated GitHub
// API client. Returns the owner, repo name, and client.
func (g *githubVerifiedPusher) newGitHubClient(
	repoURL, token string,
	insecureSkipTLSVerify bool,
) (string, string, githubVerifiedPushClient, error) {
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
			"could not extract repository owner and name from URL %q",
			repoURL,
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

	return owner, repo,
		&githubVerifiedPushClientWrapper{client: client}, nil
}

// buildCommitURL constructs a human-readable commit URL from a repository URL
// and commit SHA.
func (g *githubVerifiedPusher) buildCommitURL(
	repoURL, sha string,
) string {
	normalizedURL := urls.NormalizeGit(repoURL)
	parsedURL, err := url.Parse(normalizedURL)
	if err != nil {
		return ""
	}
	return fmt.Sprintf(
		"https://%s%s/commit/%s",
		parsedURL.Host, parsedURL.Path, sha,
	)
}
