package builtin

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/kelseyhightower/envconfig"
	"github.com/xeipuuv/gojsonschema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/gitprovider"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

const (
	stepKindGitPush = "git-push"

	// stateKeyBranch is the key used to store the branch that was pushed to in
	// the shared State.
	stateKeyBranch = "branch"

	// stateKeyCommitURL is the key used to store the URL of the commit that was
	// pushed to in the shared State.
	stateKeyCommitURL = "commitURL"
)

func init() {
	cfg := gitPusherConfigFromEnv()
	var once sync.Once
	var pusher promotion.StepRunner
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindGitPush,
			Metadata: promotion.StepRunnerMetadata{
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessCredentials,
				},
			},
			// This factory function closes over a single instance of gitPushPusher
			// so that that its mutexes are shared across all executions of this step
			// runner, which is necessary to ensure proper locking behavior.
			Value: func(caps promotion.StepRunnerCapabilities) promotion.StepRunner {
				once.Do(func() {
					pusher = newGitPusher(caps, cfg)
				})
				return pusher
			},
		},
	)
}

// gitPusherConfig holds controller-level configuration for the git-push step,
// populated from environment variables.
type gitPusherConfig struct {
	PushIntegrationPolicy git.PushIntegrationPolicy `envconfig:"GIT_PUSH_INTEGRATION_POLICY" default:"AlwaysRebase"`
}

func gitPusherConfigFromEnv() gitPusherConfig {
	cfg := gitPusherConfig{}
	envconfig.MustProcess("", &cfg)
	return cfg
}

// gitPushPusher is an implementation of the promotion.StepRunner interface that
// pushes commits and tags from a local Git repository to a remote Git repository.
type gitPushPusher struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
	cfg          gitPusherConfig
	branchMus    map[string]*sync.Mutex
	masterMu     sync.Mutex
}

// newGitPusher returns an implementation of the promotion.StepRunner interface
// that pushes commits from a local Git repository to a remote Git repository.
func newGitPusher(
	caps promotion.StepRunnerCapabilities,
	cfg gitPusherConfig,
) promotion.StepRunner {
	return &gitPushPusher{
		credsDB:      caps.CredsDB,
		cfg:          cfg,
		branchMus:    map[string]*sync.Mutex{},
		schemaLoader: getConfigSchemaLoader(stepKindGitPush),
	}
}

// Run implements the promotion.StepRunner interface.
func (g *gitPushPusher) Run(
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

// validate validates gitPusher configuration against a JSON schema.
func (g *gitPushPusher) convert(cfg promotion.Config) (builtin.GitPushConfig, error) {
	return validateAndConvert[builtin.GitPushConfig](g.schemaLoader, cfg, stepKindGitPush)
}

func (g *gitPushPusher) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.GitPushConfig,
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
		TargetBranch:      cfg.TargetBranch,
		IntegrationPolicy: g.cfg.PushIntegrationPolicy,
		Force:             cfg.Force,
	}
	// If we're supposed to generate a target branch name, do so.
	if cfg.GenerateTargetBranch {
		// TargetBranch and GenerateTargetBranch are mutually exclusive, so we're
		// never overwriting a user-specified target branch here.
		pushOpts.TargetBranch = fmt.Sprintf("kargo/promotion/%s", stepCtx.Promotion)
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
		pushOpts.Force = true
	}
	if cfg.Tag != "" {
		pushOpts.Tag = cfg.Tag
		// If we're pushing a tag, we should not attempt to integrate remote
		// changes first as tags are immutable and any existing tag with the
		// same name on the remote would cause the integration to fail.
		pushOpts.IntegrationPolicy = git.PushIntegrationPolicyNone
	}
	// Disable remote change integration when force pushing to allow
	// overwriting remote history.
	if pushOpts.Force {
		pushOpts.IntegrationPolicy = git.PushIntegrationPolicyNone
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
			// This will obtain a lock on the repo + branch before integrating remote
			// changes into the local branch and pushing the branch to the remote
			// using the GitHub API. This means retries should only ever be necessary
			// when there are multiple sharded controllers concurrently executing
			// Promotions that push to the same branch.
			return g.push(workTree, pushOpts)
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

	commitID, err := workTree.LastCommitID()
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error getting last commit ID: %w", err)
	}

	// Use the Git provider to get the commit URL, if possible. We continue
	// even if the provider or URL cannot be determined, as the push will
	// still have succeeded which is the primary goal of this step.
	gpOpts := gitprovider.Options{}
	if cfg.Provider != nil {
		gpOpts.Name = string(*cfg.Provider)
	}

	output := map[string]any{stateKeyCommit: commitID}
	if pushOpts.TargetBranch != "" {
		output[stateKeyBranch] = pushOpts.TargetBranch
	}

	gitProvider, err := gitprovider.New(workTree.URL(), &gpOpts)
	var commitURL string
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

// push obtains a repo + branch lock before integrating remote changes into the
// local branch and pushing the branch to the remote using the GitHub API. This
// helps reduce the likelihood of conflicts when multiple Promotions that push
// to the same branch are running concurrently.
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
