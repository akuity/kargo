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
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/credentials"
	ghutil "github.com/akuity/kargo/pkg/github"
	githubprov "github.com/akuity/kargo/pkg/gitprovider/github"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const (
	stepKindGitHubPush = "github-push"

	// stagingRefPrefix is the ref namespace used for temporary staging refs
	// that are invisible in the branch list.
	stagingRefPrefix = "refs/kargo/staging"
)

func init() {
	cfg := githubPusherConfigFromEnv()
	var once sync.Once
	var pusher promotion.StepRunner
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindGitHubPush,
			Metadata: promotion.StepRunnerMetadata{
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessCredentials,
				},
			},
			// This factory function closes over a single instance of githubPusher
			// so that that its mutexes are shared across all executions of this step
			// runner, which is necessary to ensure proper locking behavior.
			Value: func(caps promotion.StepRunnerCapabilities) promotion.StepRunner {
				once.Do(func() {
					pusher = newGitHubPusher(caps, cfg)
				})
				return pusher
			},
		},
	)
}

// githubPushClient is an interface for the GitHub API methods needed by
// the github-push step. It wraps go-github's GitService and
// RepositoriesService for testability.
type githubPushClient interface {
	CreateCommit(
		ctx context.Context,
		owner, repo string,
		commit github.Commit,
		opts *github.CreateCommitOptions,
	) (*github.Commit, *github.Response, error)
	CompareCommits(
		ctx context.Context,
		owner, repo, base, head string,
		opts *github.ListOptions,
	) (*github.CommitsComparison, *github.Response, error)
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

// githubPushServiceAdapter wraps go-github services to implement
// githubPushClient.
type githubPushServiceAdapter struct {
	*github.GitService
	*github.RepositoriesService
}

// githubPusherConfig holds controller-level configuration for the
// github-push step, populated from environment variables.
type githubPusherConfig struct {
	PushIntegrationPolicy  git.PushIntegrationPolicy `envconfig:"GIT_PUSH_INTEGRATION_POLICY" default:"AlwaysRebase"`
	MaxRevisions           int                       `envconfig:"GITHUB_PUSH_MAX_REVISIONS" default:"10"`
	VerifyUntrustedCommits bool                      `envconfig:"GITHUB_PUSH_VERIFY_UNTRUSTED_COMMITS" default:"false"`
}

func githubPusherConfigFromEnv() githubPusherConfig {
	cfg := githubPusherConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// githubPusher is an implementation of the promotion.StepRunner interface
// that pushes commits to a GitHub repository via the GitHub API, enabling
// GitHub's native commit verification ("Verified" badge) for commits
// signed by a trusted key.
type githubPusher struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
	cfg          githubPusherConfig
	branchMus    map[string]*sync.Mutex
	masterMu     sync.Mutex
}

// newGitHubPusher returns an implementation of the promotion.StepRunner
// interface that pushes commits to a GitHub repository via the GitHub API.
func newGitHubPusher(
	caps promotion.StepRunnerCapabilities,
	cfg githubPusherConfig,
) promotion.StepRunner {
	return &githubPusher{
		credsDB:      caps.CredsDB,
		cfg:          cfg,
		branchMus:    map[string]*sync.Mutex{},
		schemaLoader: getConfigSchemaLoader(stepKindGitHubPush),
	}
}

func (g *githubPusher) Run(
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

func (g *githubPusher) convert(
	cfg promotion.Config,
) (builtin.GitHubPushConfig, error) {
	return validateAndConvert[builtin.GitHubPushConfig](
		g.schemaLoader, cfg, stepKindGitHubPush,
	)
}

func (g *githubPusher) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.GitHubPushConfig,
) (promotion.StepResult, error) {
	logger := logging.LoggerFromContext(ctx)

	// This is kind of hacky, but we needed to load the working tree to get the
	// URL of the repository. With that in hand, we can look for applicable
	// credentials and, if found, reload the work tree with the credentials.
	path, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, fmt.Errorf(
			"error joining path %s with work dir %s: %w",
			cfg.Path, stepCtx.WorkDir, err,
		)
	}
	loadOpts := &git.LoadWorkTreeOptions{}
	workTree, err := git.LoadWorkTree(path, loadOpts)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error loading working tree from %s: %w", cfg.Path, err)
	}
	creds, err := g.credsDB.Get(
		ctx,
		stepCtx.Project,
		credentials.TypeGit,
		workTree.URL(),
	)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error getting credentials for %s: %w", workTree.URL(), err)
	}
	var token string
	switch {
	case creds == nil:
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{Err: fmt.Errorf(
				"no credentials found for %s", workTree.URL(),
			)}
	case creds.Password != "":
		loadOpts.Credentials = &git.RepoCredentials{
			Username:      creds.Username,
			Password:      creds.Password,
			SSHPrivateKey: creds.SSHPrivateKey,
		}
		token = creds.Password
	case creds.SSHPrivateKey != "":
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{Err: fmt.Errorf(
				"found SSH key for %s; this can be used for basic Git operations only "+
					"and cannot be used for GitHub API authentication; note that "+
					"authentication of any kind using SSH keys is deprecated; use a "+
					"personal access token or GitHub App instead",
				workTree.URL(),
			)}
	default:
		// This probably never happens in practice. If we get to here, creds were
		// found, but contained no token, and didn't even contain an SSH key worth
		// commenting on.
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{Err: fmt.Errorf(
				"credentials for %s are missing a password/token", workTree.URL(),
			)}
	}
	if workTree, err = git.LoadWorkTree(path, loadOpts); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error loading working tree from %s: %w", cfg.Path, err)
	}

	// Parse owner/repo for GitHub API calls.
	_, _, owner, repo, err := ghutil.ParseRepoURL(workTree.URL())
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error parsing repository URL: %w", err)
	}

	pushCfg := githubPushConfig{
		targetBranch:      cfg.TargetBranch,
		integrationPolicy: g.cfg.PushIntegrationPolicy,
		force:             cfg.Force,
		owner:             owner,
		repo:              repo,
		stagingRef:        fmt.Sprintf("%s/%s", stagingRefPrefix, stepCtx.Promotion),
	}
	// If we're supposed to generate a target branch name, do so.
	if cfg.GenerateTargetBranch {
		// TargetBranch and GenerateTargetBranch are mutually exclusive, so we're
		// never overwriting a user-specified target branch here.
		pushCfg.targetBranch = fmt.Sprintf("kargo/promotion/%s", stepCtx.Promotion)
		// Since the name of the generated branch incorporates the Promotion's
		// name, which itself incorporates a UUID, we assume this branch did not exist
		// in the remote repository prior to this Promotion. If it somehow does, the
		// only practical explanation for that would be that, for some reason, the
		// entire promotion process has restarted from step zero AFTER having
		// executed this step successfully on a prior attempt. (This can happen,
		// for instance, if the controller were restarted mid-promotion.) Enabling
		// the force push option here prevents this step from failing under those
		// circumstances, and as long as the reasonable assumption that this
		// branch is specific to this Promotion only holds, it is also safe to do this.
		pushCfg.force = true
	}
	// Disable remote change integration when force pushing to allow
	// overwriting remote history.
	if pushCfg.force {
		pushCfg.integrationPolicy = git.PushIntegrationPolicyNone
	}
	if pushCfg.targetBranch == "" {
		// If targetBranch is still empty, we want to set it to the current branch
		// because we will want to return the branch that was pushed to, but we
		// don't want to mess with the options any further.
		if pushCfg.targetBranch, err = workTree.CurrentBranch(); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("error getting current branch: %w", err)
		}
	}

	// Create the GitHub API client.
	ghClient, err := ghutil.NewClient(
		workTree.URL(),
		&ghutil.ClientOptions{
			Token:                 token,
			InsecureSkipTLSVerify: cfg.InsecureSkipTLSVerify,
		},
	)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error creating GitHub client: %w", err)
	}
	pushClient := &githubPushServiceAdapter{
		GitService:          ghClient.Git,
		RepositoriesService: ghClient.Repositories,
	}

	// Clean up the staging ref after the retry loop, regardless of outcome.
	defer g.cleanupStagingRef(
		ctx,
		pushClient,
		pushCfg.owner,
		pushCfg.repo,
		pushCfg.stagingRef,
	)

	backoff := wait.Backoff{
		// Note, the docs for this field say:
		//
		//   The remaining number of iterations in which the duration
		//   parameter may change...
		//
		// This is misleading, as it implies that the total number of attempts may
		// exceed the value of Steps and that Steps only dictates the maximum number
		// of adjustments to the interval between retries.
		//
		// Reading the implementation of retry.DefaultBackoff reveals that Steps is
		// indeed the maximum number of attempts.
		Steps:    10,
		Duration: time.Second,
		Factor:   1.5,
		Jitter:   0.5,
		Cap:      30 * time.Second,
	}
	if cfg.MaxAttempts != nil {
		backoff.Steps = int(*cfg.MaxAttempts)
	}
	if err = retry.OnError(
		backoff,
		g.isRetryableError,
		func() error {
			// This will obtain a lock on the repo + branch before integrating remote
			// changes into the local branch and pushing the branch to the remote
			// using the GitHub API. This means retries should only ever be necessary
			// when there are multiple sharded controllers concurrently executing
			// Promotions that push to the same branch.
			return g.push(ctx, workTree, pushClient, pushCfg)
		},
	); err != nil {
		if git.IsMergeConflict(err) {
			// A merge conflict requires manual resolution and no amount of retries
			// will fix that.
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
				&promotion.TerminalError{Err: err}
		}
		if errors.Is(err, git.ErrRebaseUnsafe) {
			// The integration policy prohibits merge fallback and rebase is unsafe.
			// No amount of retries will fix this.
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
				&promotion.TerminalError{Err: err}
		}
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error pushing commits to remote: %w", err)
	}

	// Sync the local branch with the remote after API replay, since the
	// replayed commits have new SHAs.
	if err = workTree.Pull(&git.PullOptions{
		Branch: pushCfg.targetBranch,
		Force:  true,
	}); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error syncing local branch after push: %w", err)
	}

	commitID, err := workTree.LastCommitID()
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error getting last commit ID: %w", err)
	}

	output := map[string]any{
		stateKeyCommit: commitID,
		stateKeyBranch: pushCfg.targetBranch,
	}

	// Use the Git provider to get the commit URL, if possible. We continue
	// even if the provider or URL cannot be determined, as the push will
	// still have succeeded which is the primary goal of this step.
	var commitURL string
	gitProvider, err := githubprov.NewProvider(workTree.URL(), nil)
	if err == nil {
		if commitURL, err = gitProvider.GetCommitURL(workTree.URL(), commitID); err != nil {
			logger.Error(err, "unable to get commit URL from Git provider")
		} else {
			output[stateKeyCommitURL] = commitURL
		}
	}

	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: output,
	}, nil
}

type githubPushConfig struct {
	force             bool
	targetBranch      string
	integrationPolicy git.PushIntegrationPolicy
	owner             string
	repo              string
	stagingRef        string
}

// push obtains a repo + branch lock before integrating remote changes into the
// local branch and pushing the branch to the remote using the GitHub API. The
// lock helps reduce the likelihood of conflicts when multiple Promotions that
// push to the same branch are running concurrently.
func (g *githubPusher) push(
	ctx context.Context,
	workTree git.WorkTree,
	client githubPushClient,
	opts githubPushConfig,
) error {
	logger := logging.LoggerFromContext(ctx)

	g.acquireBranchLock(workTree.URL(), opts.targetBranch)
	defer g.releaseBranchLock(workTree.URL(), opts.targetBranch)

	// The local branch may be behind the remote branch it tracks or may have
	// diverged from it. Start by integrating any remote changes we don't have
	// locally.
	if err := workTree.IntegrateRemoteChanges(
		&git.IntegrationOptions{
			TargetBranch:      opts.targetBranch,
			IntegrationPolicy: opts.integrationPolicy,
		},
	); err != nil {
		return fmt.Errorf(
			"error integrating remote changes from remote branch %q to local "+
				"branch %q: %w",
			opts.targetBranch, opts.targetBranch, err,
		)
	}

	// Force push the local branch to a staging ref. This gets all objects onto
	// GitHub without creating a visible branch.
	if err := workTree.Push(&git.PushOptions{
		TargetBranch: opts.stagingRef,
		Force:        true,
	}); err != nil {
		return fmt.Errorf("error pushing to staging ref %s: %w", opts.stagingRef, err)
	}

	// The head of the local branch and the remote staging ref are now guaranteed
	// to be the same commit, so we can use the ID of the commit at the head of
	// the local branch as the source commit for the upcoming comparison to the
	// target branch.
	sourceHead, err := workTree.LastCommitID()
	if err != nil {
		return fmt.Errorf("error getting source HEAD: %w", err)
	}
	logger.Debug(
		"pushed to staging ref",
		"ref", opts.stagingRef,
		"sourceHead", sourceHead,
	)

	// We need to know if the remote target branch exists.
	targetBranchExists, err := workTree.RemoteBranchExists(opts.targetBranch)
	if err != nil {
		return fmt.Errorf(
			"error checking if remote branch %s exists: %w",
			opts.targetBranch, err,
		)
	}

	// What we want to do next is identify the exact set of commits that need to
	// be replayed via the GitHub API.

	// Determine the comparison base. If the remote target branch exists, the base
	// is its HEAD. If it does not, the base is the HEAD of the remote branch
	// tracked by the current (source) branch.
	baseRefStr := "heads/" + opts.targetBranch
	if !targetBranchExists {
		var currentBranch string
		if currentBranch, err = workTree.CurrentBranch(); err != nil {
			return fmt.Errorf("error getting current branch name: %w", err)
		}
		baseRefStr = "heads/" + currentBranch
	}
	baseRef, _, err := client.GetRef(ctx, opts.owner, opts.repo, baseRefStr)
	if err != nil {
		return fmt.Errorf("error getting ref %s: %w", baseRefStr, err)
	}
	baseHead := baseRef.GetObject().GetSHA()

	// Compare base..source to identify commits to replay.
	comparison, _, err := client.CompareCommits(
		ctx,
		opts.owner,
		opts.repo,
		baseHead,
		sourceHead,
		&github.ListOptions{PerPage: 250},
	)
	if err != nil {
		return fmt.Errorf("error comparing commits %s..%s: %w", baseHead, sourceHead, err)
	}
	status := comparison.GetStatus()
	switch {
	case status == "identical":
		// Nothing at all to do.
		return nil
	case status == "ahead",
		// Normal case — fall through to replay.
		status == "behind" && opts.force,
		// Source is behind target, but force is enabled. Fall through to replay.
		// There are zero source-only commits, so replayCommits will return the
		// merge base (which equals sourceHead here), and the ref will be
		// force-updated to that — effectively moving the target backward.
		status == "diverged" && opts.force:
		// Source and target have diverged, but force is enabled. Fall through to
		// replay the source-only commits, then force-update the ref. The
		// target-only commits will be discarded.
	case status == "behind" && !opts.force:
		// Source is behind target and force is not enabled. This is retryable. The
		// integration policy may allow the source branch to be brought up to date
		// on the next attempt and then it will no longer be behind.
		return &retryableError{err: errors.New(
			"source branch is behind target branch and force pushing is not enabled",
		)}
	case status == "diverged" && !opts.force:
		// Source and target have diverged and force is not enabled. This is
		// retryable. The integration policy may allow the source branch to be
		// brought up to date on the next attempt and then the two branches will no
		// longer have diverged history.
		return &retryableError{err: errors.New(
			"source and target branches have diverged and force pushing is not enabled",
		)}
	default:
		return &promotion.TerminalError{
			Err: fmt.Errorf(
				"unexpected comparison status %q for %s..%s",
				status, baseHead, sourceHead,
			),
		}
	}

	commits := comparison.Commits
	mergeBase := comparison.MergeBaseCommit.GetSHA()
	if len(commits) > g.cfg.MaxRevisions {
		return &promotion.TerminalError{
			Err: fmt.Errorf(
				"commit range %s..%s contains %d commits, which exceeds the "+
					"configured maximum of %d",
				mergeBase, sourceHead, len(commits), g.cfg.MaxRevisions,
			),
		}
	}

	// Replay commits via GitHub API. The merge base is used as the initial
	// parent so that replayed commits are correctly rooted regardless of
	// whether the source is simply ahead or has diverged from the target.
	newTargetHead, err := g.replayCommits(
		ctx,
		client,
		opts.owner,
		opts.repo,
		commits,
		mergeBase,
		workTree,
	)
	if err != nil {
		return fmt.Errorf("error replaying commits via GitHub API: %w", err)
	}

	// Update the target branch ref.
	if err = g.upsertTargetRef(
		ctx,
		client,
		opts.owner,
		opts.repo,
		opts.targetBranch,
		newTargetHead,
		!targetBranchExists,
		opts.force,
	); err != nil {
		return fmt.Errorf("error updating target ref: %w", err)
	}

	return nil
}

// replayCommits replays each commit via the GitHub API, maintaining a SHA
// mapping table to preserve parent chain structure across the replayed
// range.
func (g *githubPusher) replayCommits(
	ctx context.Context,
	client githubPushClient,
	owner string,
	repo string,
	repoCommits []*github.RepositoryCommit,
	parentSHA string,
	workTree git.WorkTree,
) (string, error) {
	if len(repoCommits) == 0 {
		return parentSHA, nil
	}

	logger := logging.LoggerFromContext(ctx)

	shaMap := map[string]string{parentSHA: parentSHA}

	var lastSHA string
	for i, repoCommit := range repoCommits {
		if repoCommit.Commit == nil ||
			repoCommit.Commit.Tree == nil ||
			repoCommit.Commit.Tree.SHA == nil {
			return "", fmt.Errorf(
				"commit %d in range has missing tree information", i,
			)
		}

		// Get signature info for trust evaluation.
		sigInfo, err := workTree.GetCommitSignatureInfo(repoCommit.GetSHA())
		if err != nil {
			return "", fmt.Errorf(
				"error getting signature info for commit %s: %w",
				repoCommit.GetSHA(), err,
			)
		}

		// Build the replacement commit.
		newCommit := g.buildReplayCommit(repoCommit, sigInfo)
		defaultParent := parentSHA
		if lastSHA != "" {
			defaultParent = lastSHA
		}
		newCommit.Parents = g.resolveParents(repoCommit, shaMap, defaultParent)

		// Create new commit via the API.
		created, _, err := client.CreateCommit(ctx, owner, repo, newCommit, nil)
		if err != nil {
			return "", fmt.Errorf(
				"error creating commit %d/%d (original: %s): %w",
				i+1, len(repoCommits), repoCommit.GetSHA(), err,
			)
		}
		lastSHA = created.GetSHA()
		shaMap[repoCommit.GetSHA()] = lastSHA
		logger.Debug(
			"replayed commit",
			"index", i+1,
			"total", len(repoCommits),
			"original", repoCommit.GetSHA(),
			"new", lastSHA,
			"merge", len(newCommit.Parents) > 1,
			"trusted", sigInfo.Trusted,
		)
	}
	return lastSHA, nil
}

// buildReplayCommit constructs a github.Commit ready to be created via the
// API, based on the original commit and its signature info.
func (g *githubPusher) buildReplayCommit(
	original *github.RepositoryCommit,
	sigInfo *git.CommitSignatureInfo,
) github.Commit {
	commit := github.Commit{
		Tree:    &github.Tree{SHA: original.Commit.Tree.SHA},
		Message: ptr.To(original.Commit.GetMessage()),
	}
	if sigInfo.Trusted || g.cfg.VerifyUntrustedCommits {
		// Either the original commit was signed by a trusted key, or the operator
		// has opted to verify all commits regardless of trust. In both cases, we
		// withhold author/committer information from the new commit, allowing
		// GitHub to attribute it to the authenticated user and sign it with its
		// own key, which will result in a VERIFIED commit. If the original
		// commit's author differs from its signer (or is unknown, in the case of
		// untrusted commits), we add a Co-authored-by trailer to the commit
		// message to preserve that information.
		author := original.Commit.GetAuthor()
		if author != nil {
			authorName := author.GetName()
			authorEmail := author.GetEmail()
			if authorName != "" && authorEmail != "" &&
				(authorName != sigInfo.SignerName || authorEmail != sigInfo.SignerEmail) {
				commit.Message = ptr.To(fmt.Sprintf(
					"%s\n\nCo-authored-by: %s <%s>",
					commit.GetMessage(), authorName, authorEmail,
				))
			}
		}
	} else {
		// The original commit was not signed by a trusted key, so we preserve the
		// original author and committer information in the new commit. GitHub will
		// not attribute the new commit to the authenticated user, and will not sign
		// the new commit with its own key. These factors will result in an
		// UNVERIFIED commit. Provenance will remain completely intact.
		commit.Author = original.Commit.Author
		commit.Committer = original.Commit.Committer
	}
	return commit
}

// resolveParents maps a commit's parents through the SHA mapping table.
// For merge commits, each parent is mapped individually so the multi-parent
// structure is preserved.
func (g *githubPusher) resolveParents(
	repoCommit *github.RepositoryCommit,
	shaMap map[string]string,
	defaultParent string,
) []*github.Commit {
	if len(repoCommit.Parents) <= 1 {
		return []*github.Commit{{SHA: &defaultParent}}
	}
	parents := make([]*github.Commit, 0, len(repoCommit.Parents))
	for _, p := range repoCommit.Parents {
		sha := p.GetSHA()
		if mapped, ok := shaMap[sha]; ok {
			sha = mapped
		}
		parents = append(parents, &github.Commit{SHA: &sha})
	}
	return parents
}

// upsertTargetRef points the target branch at the given SHA. Returns a
// retryable error when an UpdateRef receives HTTP 422.
func (g *githubPusher) upsertTargetRef(
	ctx context.Context,
	client githubPushClient,
	owner string,
	repo string,
	targetBranch string,
	sha string,
	createBranch bool,
	force bool,
) error {
	logger := logging.LoggerFromContext(ctx)
	targetRef := "heads/" + targetBranch

	if createBranch {
		_, _, err := client.CreateRef(
			ctx,
			owner,
			repo,
			github.CreateRef{
				Ref: "refs/" + targetRef,
				SHA: sha,
			},
		)
		if err != nil {
			return fmt.Errorf("error creating ref %s at %s: %w", targetRef, sha, err)
		}
		logger.Debug("created branch ref", "ref", targetRef, "sha", sha)
		return nil
	}

	_, resp, err := client.UpdateRef(
		ctx,
		owner,
		repo,
		targetRef,
		github.UpdateRef{
			SHA:   sha,
			Force: &force,
		},
	)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnprocessableEntity {
			return &retryableError{
				err: fmt.Errorf(
					"error updating ref %s to %s (not a fast forward): %w",
					targetRef, sha, err,
				),
			}
		}
		return fmt.Errorf("error updating ref %s to %s: %w", targetRef, sha, err)
	}
	logger.Debug("updated branch ref", "ref", targetRef, "sha", sha)
	return nil
}

// cleanupStagingRef deletes the staging ref from GitHub. Errors are logged
// but do not cause the step to fail.
func (g *githubPusher) cleanupStagingRef(
	ctx context.Context,
	client githubPushClient,
	owner string,
	repo string,
	stagingRef string,
) {
	ref := strings.TrimPrefix(stagingRef, "refs/")
	if _, err := client.DeleteRef(ctx, owner, repo, ref); err != nil {
		logging.LoggerFromContext(ctx).Error(
			err,
			"error deleting staging ref (non-fatal)",
			"ref", stagingRef,
		)
	}
}

// retryableError is an error that signals the retry loop to try again.
type retryableError struct {
	err error
}

func (e *retryableError) Error() string { return e.err.Error() }
func (e *retryableError) Unwrap() error { return e.err }

func (g *githubPusher) isRetryableError(err error) bool {
	var re *retryableError
	return errors.As(err, &re)
}

func (g *githubPusher) acquireBranchLock(repoURL, branch string) {
	key := fmt.Sprintf("%s:%s", repoURL, branch)
	if _, exists := g.branchMus[key]; !exists {
		g.masterMu.Lock()
		if _, exists = g.branchMus[key]; !exists {
			g.branchMus[key] = &sync.Mutex{}
		}
		g.masterMu.Unlock()
	}
	g.branchMus[key].Lock()
}

func (g *githubPusher) releaseBranchLock(repoURL, branch string) {
	key := fmt.Sprintf("%s:%s", repoURL, branch)
	g.branchMus[key].Unlock()
}
