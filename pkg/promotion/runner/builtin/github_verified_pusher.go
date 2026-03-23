package builtin

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/google/go-github/v76/github"
	"github.com/kelseyhightower/envconfig"
	"github.com/xeipuuv/gojsonschema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/credentials"
	kargogithub "github.com/akuity/kargo/pkg/github"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/promotion"
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

	// pullPolicy values control how local commits are reconciled with the
	// remote target branch when it has advanced.
	pullPolicyMerge  = "Merge"
	pullPolicyRebase = "Rebase"
	pullPolicyFFOnly = "FFOnly"
)

// errRefUpdateConflict is returned when UpdateRef fails because the target
// branch has advanced since we last read it (HTTP 422). This signals the
// retry loop to rebase and retry.
var errRefUpdateConflict = errors.New("ref update conflict")

// isRetryableError returns true for errors that should trigger a retry:
// ref update conflicts from the GitHub API or non-fast-forward rejections
// from git push.
func (g *githubVerifiedPusher) isRetryableError(err error) bool {
	return errors.Is(err, errRefUpdateConflict) || git.IsNonFastForward(err)
}

// isGitHubHTTPStatus returns true if err is a *github.ErrorResponse with
// the given HTTP status code.
func (g *githubVerifiedPusher) isGitHubHTTPStatus(
	err error, statusCode int,
) bool {
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) {
		return ghErr.Response != nil &&
			ghErr.Response.StatusCode == statusCode
	}
	return false
}

func init() {
	var once sync.Once
	var pusher promotion.StepRunner
	envCfg := struct {
		Name           string `envconfig:"GITCLIENT_NAME" default:"Kargo"`
		Email          string `envconfig:"GITCLIENT_EMAIL" default:"no-reply@kargo.io"`
		SigningKeyType string `envconfig:"GITCLIENT_SIGNING_KEY_TYPE"`
		SigningKeyPath string `envconfig:"GITCLIENT_SIGNING_KEY_PATH"`
		MaxRevisions   int    `envconfig:"GITHUB_VERIFIED_PUSH_MAX_REVISIONS" default:"10"`
	}{}
	envconfig.MustProcess("", &envCfg)
	cfg := githubVerifiedPusherConfig{
		MaxRevisions: envCfg.MaxRevisions,
	}
	defaultGitUser := git.User{
		Name:           envCfg.Name,
		Email:          envCfg.Email,
		SigningKeyType: git.SigningKeyType(envCfg.SigningKeyType),
		SigningKeyPath: envCfg.SigningKeyPath,
	}
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindGitHubVerifiedPush,
			Metadata: promotion.StepRunnerMetadata{
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessCredentials,
				},
			},
			// This factory function closes over a single instance of
			// githubVerifiedPusher so that its mutexes are shared across
			// all executions of this step runner, which is necessary to
			// ensure proper locking behavior.
			Value: func(
				caps promotion.StepRunnerCapabilities,
			) promotion.StepRunner {
				once.Do(func() {
					pusher = newGitHubVerifiedPusher(caps, cfg, defaultGitUser)
				})
				return pusher
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

// githubVerifiedPushClientWrapper wraps a *github.GitService and
// *github.RepositoriesService to implement githubVerifiedPushClient.
type githubVerifiedPushClientWrapper struct {
	*github.GitService
	*github.RepositoriesService
}

// githubVerifiedPusherConfig holds controller-level configuration for the
// github-verified-push step, populated from environment variables.
type githubVerifiedPusherConfig struct {
	MaxRevisions int
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
	gitUser git.User,
) promotion.StepRunner {
	g := &githubVerifiedPusher{
		credsDB:      caps.CredsDB,
		schemaLoader: getConfigSchemaLoader(stepKindGitHubVerifiedPush),
		cfg:          cfg,
		gitUser:      gitUser,
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
					"no credentials found for %s;"+
						" a GitHub App installation token or personal access token is required",
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
	pullPolicy := pullPolicyMerge // secure default
	if cfg.PullPolicy != nil {
		pullPolicy = string(*cfg.PullPolicy)
	}
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

	stagingRef := fmt.Sprintf(
		"%s/%s", stagingRefPrefix, stepCtx.Promotion,
	)

	// Create the GitHub client.
	owner, repo, ghClient, err := g.newGitHubClientFn(
		workTree.URL(), token, cfg.InsecureSkipTLSVerify,
	)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusErrored,
		}, fmt.Errorf("error creating GitHub client: %w", err)
	}

	g.acquireBranchLock(workTree.URL(), targetBranch)
	defer g.releaseBranchLock(workTree.URL(), targetBranch)

	// Retry loop: each attempt reconciles with the remote target branch
	// (using the configured pullPolicy), pushes to a staging ref, replays
	// commits via the API, and updates the target branch ref. Retries are
	// triggered when the ref update fails because the target branch advanced
	// concurrently. For FFOnly, no reconciliation or retries are attempted.
	defaultSteps := 10
	if pullPolicy == pullPolicyFFOnly {
		defaultSteps = 1
	}
	backoff := wait.Backoff{
		Steps:    defaultSteps,
		Duration: time.Second,
		Factor:   1.5,
		Jitter:   0.5,
		Cap:      30 * time.Second,
	}
	if cfg.MaxAttempts != nil {
		backoff.Steps = int(*cfg.MaxAttempts)
	}

	var (
		result        promotion.StepResult
		stagingPushed bool
		cleanupOwner  = owner
		cleanupRepo   = repo
		cleanupRef    = stagingRef
		cleanupClient = ghClient
	)
	defer func() {
		if stagingPushed {
			g.cleanupStagingRef(
				ctx, cleanupClient, cleanupOwner, cleanupRepo, cleanupRef,
			)
		}
	}()

	if err = retry.OnError(
		backoff,
		g.isRetryableError,
		func() error {
			// 1. Reconcile with remote target branch using the configured
			//    pull policy. Skipped when force-pushing (overwrite
			//    semantics) and when creating a new branch (nothing to
			//    reconcile with).
			if !force && !createBranch {
				if pullErr := g.pullRemote(
					workTree, targetBranch, pullPolicy,
				); pullErr != nil {
					return pullErr
				}
			}

			// 2. Get localHead after potential rebase.
			localHead, localErr := workTree.LastCommitID()
			if localErr != nil {
				return fmt.Errorf("error getting local HEAD: %w", localErr)
			}

			// 3. Push (rebased) commits to staging ref. This gets all
			//    objects onto GitHub without creating a visible branch.
			logger.Debug(
				"pushing to staging ref",
				"ref", stagingRef,
				"localHead", localHead,
			)
			if pushErr := workTree.Push(&git.PushOptions{
				TargetBranch: stagingRef,
				Force:        true,
			}); pushErr != nil {
				return fmt.Errorf(
					"error pushing to staging ref %s: %w",
					stagingRef, pushErr,
				)
			}
			stagingPushed = true

			// 4. Read targetHead from GitHub API.
			targetHead, refErr := g.getTargetHead(
				ctx, ghClient, owner, repo,
				currentBranch, targetBranch, createBranch,
			)
			if refErr != nil {
				return refErr
			}

			logger.Debug(
				"signing revision range",
				"targetBranch", targetBranch,
				"targetHead", targetHead,
				"localHead", localHead,
			)

			// 5. Replay commits via API and update ref.
			var signErr error
			result, signErr = g.signAndUpdate(
				ctx, cfg, ghClient, owner, repo,
				targetBranch, createBranch, force,
				targetHead, localHead, workTree,
			)
			return signErr
		},
	); err != nil {
		if git.IsMergeConflict(err) {
			return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusFailed,
			}, &promotion.TerminalError{Err: err}
		}
		// Errors from early in the retry closure (pullRemote,
		// LastCommitID, staging push, getTargetHead) don't set
		// result. Default to Errored.
		if result.Status == "" {
			result.Status = kargoapi.PromotionStepStatusErrored
		}
		return result, err
	}

	// Sync local branch to match remote after API replay. The replayed
	// commits have new SHAs; without this the local working tree would
	// have stale refs. Non-fatal if it fails — the push already succeeded.
	if pullErr := workTree.ForcePull(targetBranch); pullErr != nil {
		logger.Error(
			pullErr,
			"error syncing local branch after push (non-fatal)",
			"branch", targetBranch,
		)
	}

	return result, nil
}

// getTargetHead reads the current HEAD SHA of the target branch (or source
// branch when creating a new branch) from the GitHub API.
func (g *githubVerifiedPusher) getTargetHead(
	ctx context.Context,
	client githubVerifiedPushClient,
	owner, repo, currentBranch, targetBranch string,
	createBranch bool,
) (string, error) {
	if createBranch {
		sourceRef := "heads/" + currentBranch
		ref, _, err := client.GetRef(ctx, owner, repo, sourceRef)
		if err != nil {
			return "", fmt.Errorf(
				"error getting source branch ref %s: %w",
				sourceRef, err,
			)
		}
		return ref.GetObject().GetSHA(), nil
	}
	targetRef := "heads/" + targetBranch
	ref, _, err := client.GetRef(ctx, owner, repo, targetRef)
	if err != nil {
		return "", fmt.Errorf(
			"error getting ref %s: %w", targetRef, err,
		)
	}
	return ref.GetObject().GetSHA(), nil
}

// pullRemote reconciles the local branch with the remote target branch
// according to the specified pull policy.
func (g *githubVerifiedPusher) pullRemote(
	workTree git.WorkTree,
	targetBranch, policy string,
) error {
	switch policy {
	case pullPolicyMerge:
		return workTree.PullMerge(targetBranch)
	case pullPolicyRebase:
		return workTree.PullRebase(targetBranch)
	case pullPolicyFFOnly:
		// No reconciliation — the signAndUpdate step will detect and
		// reject any divergence.
		return nil
	default:
		return fmt.Errorf("unknown pullPolicy: %q", policy)
	}
}

// signAndUpdate enumerates commits in the range targetHead..localHead,
// replays them as signed commits via the GitHub REST API, and updates the
// target branch ref to point to the final signed commit. When createBranch
// is true, a new branch is created instead of updating an existing one.
// When force is true, diverged branches are accepted and the ref update
// uses force semantics.
func (g *githubVerifiedPusher) signAndUpdate(
	ctx context.Context,
	cfg builtin.GitHubVerifiedPushConfig,
	client githubVerifiedPushClient,
	owner, repo, targetBranch string,
	createBranch, force bool,
	targetHead, localHead string,
	workTree git.WorkTree,
) (promotion.StepResult, error) {
	cmp, err := g.compareRemote(
		ctx, client, owner, repo, targetBranch,
		targetHead, localHead, force, workTree,
	)
	if err != nil {
		if promotion.IsTerminal(err) {
			return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusFailed,
			}, err
		}
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusErrored,
		}, err
	}
	if cmp.earlyResult != nil {
		return *cmp.earlyResult, nil
	}

	// Resolve the app identity for author matching.
	appName, appEmail := g.gitUser.Name, g.gitUser.Email
	if cfg.Author != nil {
		appName, appEmail = cfg.Author.Name, cfg.Author.Email
	}

	// Import the app's signing key (if configured) and compute its
	// fingerprint. The fingerprint is used during replay to identify
	// which commits were signed by the app.
	var appFingerprint string
	if cfg.Author != nil {
		var sigErr error
		appFingerprint, sigErr = g.resolveAppFingerprint(
			ctx, workTree, cfg.Author,
		)
		if sigErr != nil {
			return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusErrored,
			}, sigErr
		}
	}

	commitSigs, sigErr := g.verifyCommitSignatures(
		ctx, workTree, cmp.commits,
	)
	if sigErr != nil {
		if promotion.IsTerminal(sigErr) {
			return promotion.StepResult{
				Status: kargoapi.PromotionStepStatusFailed,
			}, sigErr
		}
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusErrored,
		}, sigErr
	}

	lastSignedSHA, err := g.replayCommits(
		ctx, client, owner, repo, cmp.commits, cmp.parentSHA,
		appName, appEmail, appFingerprint, commitSigs,
	)
	if err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusErrored,
		}, err
	}

	if err = g.updateTargetRef(
		ctx, client, owner, repo,
		targetBranch, lastSignedSHA,
		createBranch, force,
	); err != nil {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusErrored,
		}, err
	}

	commitURL := kargogithub.BuildCommitURL(workTree.URL(), lastSignedSHA)
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
	commitURL := kargogithub.BuildCommitURL(workTree.URL(), sha)
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

// newGitHubClient creates an authenticated GitHub API client for the given
// repository.
func (g *githubVerifiedPusher) newGitHubClient(
	repoURL, token string,
	insecureSkipTLSVerify bool,
) (string, string, githubVerifiedPushClient, error) {
	client, owner, repo, err := kargogithub.NewClient(
		repoURL, token, insecureSkipTLSVerify,
	)
	if err != nil {
		return "", "", nil, err
	}
	return owner, repo,
		&githubVerifiedPushClientWrapper{
			GitService:          client.Git,
			RepositoriesService: client.Repositories,
		}, nil
}

// comparisonResult holds the output of compareRemote: the resolved
// parent SHA, the commits to replay, and optionally an early result to
// return (e.g. skip or force-update) when no replay is needed.
type comparisonResult struct {
	parentSHA   string
	commits     []*github.RepositoryCommit
	earlyResult *promotion.StepResult
}

// compareRemote calls the GitHub Compare API and interprets the
// result. It resolves the parent SHA for the first replayed commit,
// validates the commit list, and returns early results for skip and
// force-update cases.
func (g *githubVerifiedPusher) compareRemote(
	ctx context.Context,
	client githubVerifiedPushClient,
	owner, repo, targetBranch string,
	targetHead, localHead string,
	force bool,
	workTree git.WorkTree,
) (*comparisonResult, error) {
	logger := logging.LoggerFromContext(ctx)

	comparison, _, err := client.CompareCommits(
		ctx, owner, repo, targetHead, localHead, nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
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

	parentSHA := targetHead

	switch status {
	case compareStatusAhead:
		// Expected: target is behind local head.
	case compareStatusIdentical:
		return &comparisonResult{
			earlyResult: &promotion.StepResult{
				Status: kargoapi.PromotionStepStatusSkipped,
				Message: fmt.Sprintf(
					"no revisions to sign: %s and %s are identical",
					targetHead, localHead,
				),
				Output: map[string]any{
					stateKeyCommit: targetHead,
					stateKeyBranch: targetBranch,
				},
			},
		}, nil
	case compareStatusDiverged, compareStatusBehind:
		if !force {
			return nil, &promotion.TerminalError{
				Err: fmt.Errorf(
					"cannot sign revision range %s..%s: "+
						"comparison status is %q "+
						"(target branch may have diverged)",
					targetHead, localHead, status,
				),
			}
		}
		if status == compareStatusDiverged {
			mergeBase := comparison.GetMergeBaseCommit()
			if mergeBase == nil {
				return nil, fmt.Errorf(
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
		return nil, &promotion.TerminalError{
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
			result, err := g.forceUpdateRef(
				ctx, client, owner, repo,
				targetBranch, localHead, workTree,
			)
			return &comparisonResult{earlyResult: &result}, err
		}
		return &comparisonResult{
			earlyResult: &promotion.StepResult{
				Status:  kargoapi.PromotionStepStatusSkipped,
				Message: "no revisions to sign in range",
				Output: map[string]any{
					stateKeyCommit: targetHead,
					stateKeyBranch: targetBranch,
				},
			},
		}, nil
	}

	if len(commits) > g.cfg.MaxRevisions {
		return nil, &promotion.TerminalError{
			Err: fmt.Errorf(
				"revision range %s..%s contains %d revisions, "+
					"which exceeds the maximum of %d "+
					"(configurable via "+
					"GITHUB_VERIFIED_PUSH_MAX_REVISIONS env var)",
				targetHead, localHead, len(commits), g.cfg.MaxRevisions,
			),
		}
	}

	return &comparisonResult{
		parentSHA: parentSHA,
		commits:   commits,
	}, nil
}

// replayCommits replays each commit via the GitHub API. Commits that are
// "app-authored" (author==committer, matches configured app identity, and
// signed by the app's key when a fingerprint is provided) are created without
// explicit author/committer fields, allowing the GitHub App to sign them as
// "Verified". All other commits preserve their original author and committer.
//
// It is merge-aware: merge commits are created with their original
// multi-parent structure preserved via a SHA mapping table.
func (g *githubVerifiedPusher) replayCommits(
	ctx context.Context,
	client githubVerifiedPushClient,
	owner, repo string,
	commits []*github.RepositoryCommit,
	parentSHA string,
	appName, appEmail, appFingerprint string,
	commitSigs map[string]git.CommitSignatureInfo,
) (string, error) {
	logger := logging.LoggerFromContext(ctx)

	shaMap := map[string]string{parentSHA: parentSHA}
	var lastSignedSHA string
	for i, rc := range commits {
		if rc.Commit == nil ||
			rc.Commit.Tree == nil ||
			rc.Commit.Tree.SHA == nil {
			return "", fmt.Errorf(
				"revision %d in range has missing tree information", i,
			)
		}

		defaultParent := parentSHA
		if lastSignedSHA != "" {
			defaultParent = lastSignedSHA
		}
		parents := g.resolveParents(rc, shaMap, defaultParent)

		message := rc.Commit.GetMessage()

		commit := github.Commit{
			Message: &message,
			Tree:    &github.Tree{SHA: rc.Commit.Tree.SHA},
			Parents: parents,
		}

		// Determine whether this is an app-authored commit.
		appAuthored := g.isAppAuthored(
			rc, appName, appEmail, appFingerprint, commitSigs,
		)
		if !appAuthored {
			// Preserve original author. The commit will not receive
			// GitHub's "Verified" badge. Committer is always left nil
			// so the authenticated identity (GitHub App) is used — it
			// is the actual committer in all cases.
			commit.Author = rc.Commit.Author
		}

		newCommit, _, createErr := client.CreateCommit(
			ctx, owner, repo, commit, nil,
		)
		if createErr != nil {
			return "", fmt.Errorf(
				"error creating revision %d/%d "+
					"(original: %s): %w",
				i+1, len(commits), rc.GetSHA(), createErr,
			)
		}
		lastSignedSHA = newCommit.GetSHA()
		shaMap[rc.GetSHA()] = lastSignedSHA
		logger.Debug(
			"created revision",
			"index", i+1,
			"total", len(commits),
			"original", rc.GetSHA(),
			"new", lastSignedSHA,
			"merge", len(parents) > 1,
			"appAuthored", appAuthored,
		)
	}
	return lastSignedSHA, nil
}

// isAppAuthored returns true when the commit was authored by the configured
// app identity. The check requires: author == committer, author matches the
// app's name/email, and (when a fingerprint is provided) the commit's signing
// key fingerprint matches.
func (g *githubVerifiedPusher) isAppAuthored(
	rc *github.RepositoryCommit,
	appName, appEmail, appFingerprint string,
	commitSigs map[string]git.CommitSignatureInfo,
) bool {
	author := rc.Commit.GetAuthor()
	committer := rc.Commit.GetCommitter()
	if author == nil || committer == nil {
		return false
	}
	sameIdentity := author.GetName() == committer.GetName() &&
		author.GetEmail() == committer.GetEmail()
	matchesApp := author.GetName() == appName &&
		author.GetEmail() == appEmail
	sigVerified := appFingerprint == "" ||
		commitSigs[rc.GetSHA()].Fingerprint == appFingerprint
	return sameIdentity && matchesApp && sigVerified
}

// updateTargetRef points the target branch at the given SHA. When
// createBranch is true, a new ref is created; otherwise the existing ref
// is updated. Returns errRefUpdateConflict when an UpdateRef receives
// HTTP 422, signaling the retry loop to try again.
func (g *githubVerifiedPusher) updateTargetRef(
	ctx context.Context,
	client githubVerifiedPushClient,
	owner, repo, targetBranch, sha string,
	createBranch, force bool,
) error {
	logger := logging.LoggerFromContext(ctx)
	targetRef := "heads/" + targetBranch

	if createBranch {
		_, _, err := client.CreateRef(
			ctx, owner, repo,
			github.CreateRef{
				Ref: "refs/" + targetRef,
				SHA: sha,
			},
		)
		if err != nil {
			return fmt.Errorf(
				"error creating ref %s at %s: %w",
				targetRef, sha, err,
			)
		}
		logger.Debug("created branch ref", "ref", targetRef, "sha", sha)
		return nil
	}

	_, _, err := client.UpdateRef(
		ctx, owner, repo, targetRef,
		github.UpdateRef{
			SHA:   sha,
			Force: &force,
		},
	)
	if err != nil {
		if g.isGitHubHTTPStatus(err, http.StatusUnprocessableEntity) {
			return fmt.Errorf(
				"error updating ref %s to %s: %w",
				targetRef, sha, errRefUpdateConflict,
			)
		}
		return fmt.Errorf(
			"error updating ref %s to %s: %w",
			targetRef, sha, err,
		)
	}
	logger.Debug("updated branch ref", "ref", targetRef, "sha", sha)
	return nil
}

// resolveParents maps a commit's original parent SHAs to their replayed
// equivalents using the provided SHA map. Parents not in the map are assumed
// to already exist on the remote and are used as-is. This preserves
// multi-parent (merge) commit structure through the API replay.
//
// When the commit has no parent information (e.g. sparse API response),
// defaultParent is used as the single parent to maintain linear chain
// behavior.
func (g *githubVerifiedPusher) resolveParents(
	rc *github.RepositoryCommit,
	shaMap map[string]string,
	defaultParent string,
) []*github.Commit {
	origParents := rc.Parents
	if len(origParents) == 0 && rc.Commit != nil {
		origParents = rc.Commit.Parents
	}
	if len(origParents) == 0 {
		return []*github.Commit{{SHA: &defaultParent}}
	}
	parents := make([]*github.Commit, 0, len(origParents))
	for _, p := range origParents {
		sha := p.GetSHA()
		if mapped, ok := shaMap[sha]; ok {
			sha = mapped
		}
		parents = append(parents, &github.Commit{SHA: &sha})
	}
	return parents
}

// resolveAppFingerprint imports the app's signing key into the work tree's
// GPG keyring and returns its fingerprint. The fingerprint is used during
// replay to identify commits signed by the app's key.
func (g *githubVerifiedPusher) resolveAppFingerprint(
	ctx context.Context,
	workTree git.WorkTree,
	author *builtin.GitHubVerifiedPushConfigAuthor,
) (string, error) {
	if author.SigningKey == "" {
		return "", nil
	}
	logger := logging.LoggerFromContext(ctx)
	logger.Debug(
		"importing author signing key into work tree keyring",
	)
	fingerprint, err := workTree.ImportGPGKey(author.SigningKey)
	if err != nil {
		return "", fmt.Errorf(
			"error importing author signing key: %w", err,
		)
	}
	return fingerprint, nil
}

// verifyCommitSignatures checks GPG signatures on local commits before they
// are replayed via the GitHub API. Returns the signature status map and any
// error. A terminal error is returned for bad ("B") or revoked ("R")
// signatures.
func (g *githubVerifiedPusher) verifyCommitSignatures(
	ctx context.Context,
	workTree git.WorkTree,
	commits []*github.RepositoryCommit,
) (map[string]git.CommitSignatureInfo, error) {
	logger := logging.LoggerFromContext(ctx)

	// Collect SHAs to verify.
	var shas []string
	for _, rc := range commits {
		if sha := rc.GetSHA(); sha != "" {
			shas = append(shas, sha)
		}
	}
	if len(shas) == 0 {
		return nil, nil
	}

	logger.Debug(
		"verifying commit signatures",
		"numCommits", len(shas),
	)

	statuses, err := workTree.CommitSignatureStatuses(shas)
	if err != nil {
		return nil, fmt.Errorf(
			"error checking commit signatures: %w", err,
		)
	}

	for _, sha := range shas {
		info := statuses[sha]
		switch info.Status {
		case "G", "U":
			logger.Debug(
				"commit signature verified",
				"commit", sha,
				"fingerprint", info.Fingerprint,
				"signer", info.Signer,
			)
		case "N", "", "X", "Y", "E":
			logger.Debug(
				"commit not signed by trusted key, propagating",
				"commit", sha,
				"status", info.Status,
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
