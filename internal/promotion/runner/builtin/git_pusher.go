package builtin

import (
	"context"
	"fmt"
	"sync"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

// stateKeyBranch is the key used to store the branch that was pushed to in the
// shared State.
const stateKeyBranch = "branch"

// gitPushPusher is an implementation of the promotion.StepRunner interface that
// pushes commits from a local Git repository to a remote Git repository.
type gitPushPusher struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
	branchMus    map[string]*sync.Mutex
	masterMu     sync.Mutex
}

// newGitPusher returns an implementation of the promotion.StepRunner interface
// that pushes commits from a local Git repository to a remote Git repository.
func newGitPusher(credsDB credentials.Database) promotion.StepRunner {
	r := &gitPushPusher{
		credsDB:   credsDB,
		branchMus: map[string]*sync.Mutex{},
	}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (g *gitPushPusher) Name() string {
	return "git-push"
}

// Run implements the promotion.StepRunner interface.
func (g *gitPushPusher) Run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
) (promotion.StepResult, error) {
	if err := g.validate(stepCtx.Config); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, err
	}
	cfg, err := promotion.ConfigToStruct[builtin.GitPushConfig](stepCtx.Config)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("could not convert config into git-push config: %w", err)
	}
	return g.run(ctx, stepCtx, cfg)
}

// validate validates gitPusher configuration against a JSON schema.
func (g *gitPushPusher) validate(cfg promotion.Config) error {
	return validate(g.schemaLoader, gojsonschema.NewGoLoader(cfg), "git-push")
}

func (g *gitPushPusher) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.GitPushConfig,
) (promotion.StepResult, error) {
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
	if creds != nil {
		loadOpts.Credentials = &git.RepoCredentials{
			Username:      creds.Username,
			Password:      creds.Password,
			SSHPrivateKey: creds.SSHPrivateKey,
		}
	}
	if workTree, err = git.LoadWorkTree(path, loadOpts); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error loading working tree from %s: %w", cfg.Path, err)
	}
	pushOpts := &git.PushOptions{
		// Start with whatever was specified in the config, which may be empty.
		TargetBranch: cfg.TargetBranch,
		// Attempt to rebase on top of the state of the remote branch to help
		// avoid conflicts.
		PullRebase: true,
	}
	// If we're supposed to generate a target branch name, do so.
	if cfg.GenerateTargetBranch {
		// TargetBranch and GenerateTargetBranch are mutually exclusive, so we're
		// never overwriting a user-specified target branch here.
		pushOpts.TargetBranch = fmt.Sprintf("kargo/promotion/%s", stepCtx.Promotion)
		pushOpts.Force = true
	}
	if pushOpts.TargetBranch == "" {
		// If targetBranch is still empty, we want to set it to the current branch
		// because we will want to return the branch that was pushed to, but we
		// don't want to mess with the options any further.
		if pushOpts.TargetBranch, err = workTree.CurrentBranch(); err != nil {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
				fmt.Errorf("error getting current branch: %w", err)
		}
	}

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
		git.IsNonFastForward,
		func() error {
			// This will obtain a lock on the repo + branch before performing a
			// pull/rebase + push. This means retries should only ever be necessary
			// when there are multiple sharded controllers concurrently executing
			// Promotions that push to the same branch.
			return g.push(workTree, pushOpts)
		},
	); err != nil {
		if git.IsMergeConflict(err) {
			// Special case: A merge conflict requires manual resolution and no amount
			// of retries will fix that.
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
				&promotion.TerminalError{Err: err}
		}
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error pushing commits to remote: %w", err)
	}

	commitID, err := workTree.LastCommitID()
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error getting last commit ID: %w", err)
	}
	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: map[string]any{
			stateKeyBranch: pushOpts.TargetBranch,
			stateKeyCommit: commitID,
		},
	}, nil
}

// push obtains a repo + branch lock before pushing to the remote. This helps
// reduce the likelihood of conflicts when multiple Promotions that push to
// the same branch are running concurrently.
func (g *gitPushPusher) push(workTree git.WorkTree, pushOpts *git.PushOptions) error {
	branchKey := g.getBranchKey(workTree.URL(), pushOpts.TargetBranch)
	if _, exists := g.branchMus[branchKey]; !exists {
		g.masterMu.Lock()
		// Double-check to make sure it wasn't created while we were waiting for the
		// lock.
		if _, exists = g.branchMus[branchKey]; !exists {
			g.branchMus[branchKey] = &sync.Mutex{}
		}
		g.masterMu.Unlock()
	}
	g.branchMus[branchKey].Lock()
	defer g.branchMus[branchKey].Unlock()
	return workTree.Push(pushOpts)
}

func (g *gitPushPusher) getBranchKey(repoURL, branch string) string {
	return fmt.Sprintf("%s:%s", repoURL, branch)
}
