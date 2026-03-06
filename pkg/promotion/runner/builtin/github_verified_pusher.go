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
	compareStatusDiverged  = "diverged"
	compareStatusBehind    = "behind"
	compareStatusIdentical = "identical"
)

func init() {
	cfg := githubVerifiedPusherConfig{}
	envconfig.MustProcess("", &cfg)
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindGitHubVerifiedPush,
			Metadata: promotion.StepRunnerMetadata{
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessCredentials,
				},
			},
			Value: func(
				caps promotion.StepRunnerCapabilities,
			) promotion.StepRunner {
				return newGitHubVerifiedPusher(caps, cfg)
			},
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
	CreateRef(
		ctx context.Context,
		owner, repo string,
		ref github.CreateRef,
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

func (w *githubVerifiedPushClientWrapper) CreateRef(
	ctx context.Context,
	owner, repo string,
	ref github.CreateRef,
) (*github.Reference, *github.Response, error) {
	return w.client.Git.CreateRef(ctx, owner, repo, ref)
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
	schemaLoader      gojsonschema.JSONLoader
	credsDB           credentials.Database
	cfg               githubVerifiedPusherConfig
	gitUser           git.User
	branchMus         map[string]*sync.Mutex
	masterMu          sync.Mutex
	loadWorkTreeFn    func(string, *git.LoadWorkTreeOptions) (git.WorkTree, error)
	newGitHubClientFn func(
		repoURL, token string,
		insecureSkipTLSVerify bool,
	) (string, string, githubVerifiedPushClient, error)
}

// newGitHubVerifiedPusher returns an implementation of the
// promotion.StepRunner interface that pushes local commits to a GitHub
// repository as verified commits.
func newGitHubVerifiedPusher(
	caps promotion.StepRunnerCapabilities,
	cfg githubVerifiedPusherConfig,
) promotion.StepRunner {
	g := &githubVerifiedPusher{
		credsDB:      caps.CredsDB,
		schemaLoader: getConfigSchemaLoader(stepKindGitHubVerifiedPush),
		cfg:          cfg,
		gitUser:      gitUserFromEnv(),
		branchMus:    map[string]*sync.Mutex{},
	}
	g.loadWorkTreeFn = git.LoadWorkTree
	g.newGitHubClientFn = g.newGitHubClient
	return g
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
	workTree, err := g.loadWorkTreeFn(path, loadOpts)
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
	if workTree, err = g.loadWorkTreeFn(path, loadOpts); err != nil {
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

	// Resolve the current and target branches.
	currentBranch, err := workTree.CurrentBranch()
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusErrored,
		}, fmt.Errorf("error getting current branch: %w", err)
	}
	targetBranch := cfg.TargetBranch
	createBranch := false
	force := cfg.Force
	if cfg.GenerateTargetBranch {
		targetBranch = fmt.Sprintf(
			"kargo/promotion/%s", stepCtx.Promotion,
		)
		createBranch = true
		// Since the generated branch name incorporates the Promotion's
		// name (which includes a UUID), the only practical reason for
		// the branch to already exist is a promotion restart. Force
		// pushing is safe and prevents failures under those
		// circumstances. This matches the git-push step's behavior.
		force = true
	}
	if targetBranch == "" {
		targetBranch = currentBranch
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
	owner, repo, ghClient, err := g.newGitHubClientFn(
		workTree.URL(), token, cfg.InsecureSkipTLSVerify,
	)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusErrored,
		}, fmt.Errorf("error creating GitHub client: %w", err)
	}

	// Clean up the staging ref regardless of outcome. This defer is only
	// reached after the push to the staging ref has succeeded.
	defer g.cleanupStagingRef(ctx, ghClient, owner, repo, stagingRef)

	// Get the base SHA for comparison. For a new branch, use the source
	// (current) branch HEAD; otherwise use the target branch HEAD.
	var targetHead string
	if createBranch {
		sourceRef := "heads/" + currentBranch
		ref, _, refErr := ghClient.GetRef(ctx, owner, repo, sourceRef)
		if refErr != nil {
			return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, fmt.Errorf(
					"error getting source branch ref %s: %w",
					sourceRef, refErr,
				)
		}
		targetHead = ref.GetObject().GetSHA()
	} else {
		targetRef := "heads/" + targetBranch
		ref, _, refErr := ghClient.GetRef(ctx, owner, repo, targetRef)
		if refErr != nil {
			return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, fmt.Errorf(
					"error getting ref %s: %w", targetRef, refErr,
				)
		}
		targetHead = ref.GetObject().GetSHA()
	}

	logger.Debug(
		"signing revision range",
		"targetBranch", targetBranch,
		"targetHead", targetHead,
		"localHead", localHead,
	)

	// Enumerate commits to sign using the Compare API.
	result, err := g.signAndUpdate(
		ctx, ghClient, owner, repo,
		targetBranch, createBranch, force, targetHead, localHead,
		workTree,
	)
	if err != nil {
		return result, err
	}

	return result, nil
}

// signAndUpdate enumerates commits in the range targetHead..localHead,
// replays them as signed commits via the GitHub REST API, and updates the
// target branch ref to point to the final signed commit. When createBranch
// is true, a new branch is created instead of updating an existing one.
// When force is true, diverged branches are accepted and the ref update
// uses force semantics.
func (g *githubVerifiedPusher) signAndUpdate(
	ctx context.Context,
	client githubVerifiedPushClient,
	owner, repo, targetBranch string,
	createBranch, force bool,
	targetHead, localHead string,
	workTree git.WorkTree,
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

	// parentSHA is the parent for the first replayed commit.
	parentSHA := targetHead

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
	case compareStatusDiverged, compareStatusBehind:
		if !force {
			return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusFailed,
				}, &promotion.TerminalError{
					Err: fmt.Errorf(
						"cannot sign revision range %s..%s: "+
							"comparison status is %q "+
							"(target branch may have diverged; "+
							"use force to overwrite)",
						targetHead, localHead, status,
					),
				}
		}
		// When force-pushing a diverged branch, start replay from the
		// merge base so the remote branch ends up with the local
		// history only, matching git push --force semantics.
		if status == compareStatusDiverged {
			mergeBase := comparison.GetMergeBaseCommit()
			if mergeBase == nil {
				return promotion.StepResult{
						Status: kargoapi.PromotionStepStatusErrored,
					}, fmt.Errorf(
						"cannot determine merge base for %s..%s",
						targetHead, localHead,
					)
			}
			parentSHA = mergeBase.GetSHA()
			logger.Debug(
				"force push: using merge base as parent",
				"mergeBase", parentSHA,
			)
		}
	default:
		return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusFailed,
			}, &promotion.TerminalError{
				Err: fmt.Errorf(
					"cannot sign revision range %s..%s: "+
						"comparison status is %q",
					targetHead, localHead, status,
				),
			}
	}

	commits := comparison.Commits
	if len(commits) == 0 {
		if force && status == compareStatusBehind {
			// Local is behind target — force-update the ref to the
			// local HEAD without replaying any commits.
			return g.forceUpdateRef(
				ctx, client, owner, repo,
				targetBranch, localHead, workTree,
			)
		}
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

	// Verify GPG signatures on local commits before replaying them. This
	// catches tampering of signed commits while allowing unsigned commits
	// through. The returned map (nil when signing is not configured) tells
	// the replay loop which commits to re-sign via the GitHub App.
	sigStatuses, err := g.verifyCommitSignatures(ctx, workTree, commits)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusFailed,
		}, err
	}

	// Replay each revision via the API. Only commits signed by Kargo's
	// trusted key(s) (status "G") are re-signed by the GitHub App; all
	// others preserve original authorship.
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

		commit := github.Commit{
			Message: &message,
			Tree:    &github.Tree{SHA: rc.Commit.Tree.SHA},
			Parents: []*github.Commit{{SHA: &parentSHA}},
		}

		// Only let GitHub sign commits that were GPG-signed by Kargo.
		// All other commits preserve original authorship.
		sigStatus := ""
		if sigStatuses != nil {
			sigStatus = sigStatuses[rc.GetSHA()].Status
		}
		shouldSign := sigStatus == "G"
		if !shouldSign {
			commit.Author = rc.Commit.Author
			commit.Committer = rc.Commit.Committer
		} else if author := rc.Commit.GetAuthor(); author != nil {
			// Credit the original author in the re-signed commit.
			// GitHub renders Co-authored-by trailers as linked avatars.
			if name, email := author.GetName(), author.GetEmail(); name != "" && email != "" {
				message = appendCoAuthoredBy(message, name, email)
				commit.Message = &message
			}
		}

		newCommit, _, createErr := client.CreateCommit(
			ctx, owner, repo, commit, nil,
		)
		if createErr != nil {
			return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, fmt.Errorf(
					"error creating revision %d/%d "+
						"(original: %s): %w",
					i+1, len(commits), rc.GetSHA(), createErr,
				)
		}
		lastSignedSHA = newCommit.GetSHA()
		parentSHA = lastSignedSHA
		logger.Debug(
			"created revision",
			"index", i+1,
			"total", len(commits),
			"original", rc.GetSHA(),
			"new", lastSignedSHA,
			"signed", shouldSign,
		)
	}

	// Point the target branch at the final signed commit.
	targetRef := "heads/" + targetBranch
	if createBranch {
		// Create a new branch ref.
		_, _, err = client.CreateRef(
			ctx, owner, repo,
			github.CreateRef{
				Ref: "refs/" + targetRef,
				SHA: lastSignedSHA,
			},
		)
		if err != nil {
			return promotion.StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, fmt.Errorf(
					"error creating ref %s at %s: %w",
					targetRef, lastSignedSHA, err,
				)
		}
		logger.Debug(
			"created branch ref",
			"ref", targetRef,
			"sha", lastSignedSHA,
		)
	} else {
		_, _, err = client.UpdateRef(
			ctx, owner, repo, targetRef,
			github.UpdateRef{
				SHA:   lastSignedSHA,
				Force: &force,
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
	}

	commitURL := buildCommitURL(workTree.URL(), lastSignedSHA)

	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: map[string]any{
			stateKeyCommit:    lastSignedSHA,
			stateKeyCommitURL: commitURL,
			stateKeyBranch:    targetBranch,
		},
	}, nil
}

// forceUpdateRef force-updates an existing branch ref without replaying any
// commits. This handles the "behind" case when force is enabled: the local
// HEAD has no new commits relative to the target, but we still want the
// remote branch to point to the local HEAD.
func (g *githubVerifiedPusher) forceUpdateRef(
	ctx context.Context,
	client githubVerifiedPushClient,
	owner, repo, targetBranch, sha string,
	workTree git.WorkTree,
) (promotion.StepResult, error) {
	logger := logging.LoggerFromContext(ctx)
	targetRef := "heads/" + targetBranch
	force := true
	_, _, err := client.UpdateRef(
		ctx, owner, repo, targetRef,
		github.UpdateRef{
			SHA:   sha,
			Force: &force,
		},
	)
	if err != nil {
		return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusErrored,
			}, fmt.Errorf(
				"error force-updating ref %s to %s: %w",
				targetRef, sha, err,
			)
	}
	logger.Debug(
		"force-updated branch ref",
		"ref", targetRef,
		"sha", sha,
	)
	commitURL := buildCommitURL(workTree.URL(), sha)
	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: map[string]any{
			stateKeyCommit:    sha,
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

// parseGitHubRepoURL extracts the scheme, host, owner, and repo name from a
// Git repository URL.
func parseGitHubRepoURL(
	repoURL string,
) (scheme, host, owner, repo string, err error) {
	u, err := url.Parse(urls.NormalizeGit(repoURL))
	if err != nil {
		return "", "", "", "",
			fmt.Errorf("error parsing repository URL %q: %w", repoURL, err)
	}
	scheme = u.Scheme
	if scheme != "https" && scheme != "http" {
		scheme = "https"
	}
	host = u.Host
	path := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", "", "", "", fmt.Errorf(
			"could not extract repository owner and name from URL %q",
			repoURL,
		)
	}
	return scheme, host, parts[0], parts[1], nil
}

// newGitHubClient creates an authenticated GitHub API client for the given
// repository.
func (g *githubVerifiedPusher) newGitHubClient(
	repoURL, token string,
	insecureSkipTLSVerify bool,
) (string, string, githubVerifiedPushClient, error) {
	scheme, host, owner, repo, err := parseGitHubRepoURL(repoURL)
	if err != nil {
		return "", "", nil, err
	}

	httpClient := cleanhttp.DefaultClient()
	if insecureSkipTLSVerify {
		transport := cleanhttp.DefaultTransport()
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint: gosec
		}
		httpClient.Transport = transport
	}

	client := github.NewClient(httpClient)
	if host != "github.com" {
		baseURL := fmt.Sprintf("%s://%s", scheme, host)
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
func buildCommitURL(repoURL, sha string) string {
	_, host, _, _, err := parseGitHubRepoURL(repoURL)
	if err != nil {
		return ""
	}
	u, err := url.Parse(urls.NormalizeGit(repoURL))
	if err != nil {
		return ""
	}
	return fmt.Sprintf("https://%s%s/commit/%s", host, u.Path, sha)
}

// verifyCommitSignatures verifies GPG signatures on local commits before they
// are replayed via the GitHub API. When the controller is configured with a
// signing key (GITCLIENT_SIGNING_KEY_PATH), this method uses the WorkTree's
// GPG keyring (populated by git-clone's setupAuthor) to check signatures.
//
// Only commits signed by Kargo's trusted key(s) (status "G") are considered
// authentic and will be re-signed. Since commits are created during
// promotion, the signing key cannot have expired between signing and
// verification, so expired statuses ("X"/"Y") are treated as unexpected
// and propagated without re-signing. Unsigned commits, commits signed by
// untrusted keys ("U"), and commits signed by unknown keys ("E") are
// propagated without error. Commits with invalid signatures ("B") or
// revoked keys ("R") cause a terminal error.
func (g *githubVerifiedPusher) verifyCommitSignatures(
	ctx context.Context,
	workTree git.WorkTree,
	commits []*github.RepositoryCommit,
) (map[string]git.CommitSignatureInfo, error) {
	if g.gitUser.SigningKeyPath == "" {
		return nil, nil
	}

	logger := logging.LoggerFromContext(ctx)

	// Collect non-empty SHAs. RepositoryCommit.SHA is a *string that could
	// deserialize to nil from a malformed API response; skip defensively.
	var shas []string
	for _, rc := range commits {
		if sha := rc.GetSHA(); sha != "" {
			shas = append(shas, sha)
		} else {
			logger.Debug("skipping commit with empty SHA")
		}
	}
	if len(shas) == 0 {
		return nil, nil
	}

	logger.Debug(
		"verifying commit signatures",
		"signingKeyPath", g.gitUser.SigningKeyPath,
		"numCommits", len(shas),
	)

	// Single batch call to get signature info for all commits.
	statuses, err := workTree.CommitSignatureStatuses(shas)
	if err != nil {
		return nil, fmt.Errorf(
			"error checking commit signatures: %w", err,
		)
	}

	for _, sha := range shas {
		info := statuses[sha]
		switch info.Status {
		case "G":
			// Good signature from Kargo's trusted key(s).
			logger.Debug(
				"commit signature verified",
				"commit", sha,
				"status", info.Status,
				"keyID", info.KeyID,
				"signer", info.Signer,
			)
		case "N", "", "U", "X", "Y":
			// Unsigned, untrusted, or expired — propagated without
			// re-signing. "U" means the key is in the keyring but not
			// marked as trusted. "X"/"Y" indicate an expired signature
			// or key, which is unexpected for freshly-created commits.
			logger.Debug(
				"commit not signed by trusted key, propagating",
				"commit", sha,
				"status", info.Status,
			)
		case "E":
			// Signed by a key not in Kargo's keyring — not our concern.
			logger.Debug(
				"commit signed by unknown key, propagating",
				"commit", sha,
				"keyID", info.KeyID,
				"signer", info.Signer,
			)
		case "B":
			return nil, &promotion.TerminalError{
				Err: fmt.Errorf(
					"commit %s has a bad GPG signature", sha,
				),
			}
		case "R":
			return nil, &promotion.TerminalError{
				Err: fmt.Errorf(
					"commit %s was signed with a revoked key", sha,
				),
			}
		default:
			return nil, &promotion.TerminalError{
				Err: fmt.Errorf(
					"commit %s GPG signature verification failed "+
						"(status: %s)",
					sha, info.Status,
				),
			}
		}
	}

	logger.Debug("all commit signatures verified")
	return statuses, nil
}

// appendCoAuthoredBy appends a Co-authored-by trailer to a commit message,
// adding a blank separator line before the trailer if needed.
func appendCoAuthoredBy(message, name, email string) string {
	trailer := fmt.Sprintf("Co-authored-by: %s <%s>", name, email)
	if strings.HasSuffix(message, "\n\n") {
		return message + trailer
	}
	if strings.HasSuffix(message, "\n") {
		return message + "\n" + trailer
	}
	return message + "\n\n" + trailer
}
