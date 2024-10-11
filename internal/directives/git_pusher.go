package directives

import (
	"context"
	"fmt"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
)

const branchKey = "branch"

func init() {
	builtins.RegisterPromotionStepRunner(
		newGitPusher(),
		&StepRunnerPermissions{AllowCredentialsDB: true},
	)
}

// gitPushPusher is an implementation of the PromotionStepRunner interface that
// pushes commits from a local Git repository to a remote Git repository.
type gitPushPusher struct {
	schemaLoader gojsonschema.JSONLoader
}

// newGitPusher returns an implementation of the PromotionStepRunner interface
// that pushes commits from a local Git repository to a remote Git repository.
func newGitPusher() PromotionStepRunner {
	r := &gitPushPusher{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner interface.
func (g *gitPushPusher) Name() string {
	return "git-push"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (g *gitPushPusher) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	if err := g.validate(stepCtx.Config); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}
	cfg, err := configToStruct[GitPushConfig](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("could not convert config into git-push config: %w", err)
	}
	return g.runPromotionStep(ctx, stepCtx, cfg)
}

// validate validates gitPusher configuration against a JSON schema.
func (g *gitPushPusher) validate(cfg Config) error {
	return validate(g.schemaLoader, gojsonschema.NewGoLoader(cfg), "git-push")
}

func (g *gitPushPusher) runPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	cfg GitPushConfig,
) (PromotionStepResult, error) {
	// This is kind of hacky, but we needed to load the working tree to get the
	// URL of the repository. With that in hand, we can look for applicable
	// credentials and, if found, reload the work tree with the credentials.
	path, err := securejoin.SecureJoin(stepCtx.WorkDir, cfg.Path)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, fmt.Errorf(
			"error joining path %s with work dir %s: %w",
			cfg.Path, stepCtx.WorkDir, err,
		)
	}
	loadOpts := &git.LoadWorkTreeOptions{}
	workTree, err := git.LoadWorkTree(path, loadOpts)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error loading working tree from %s: %w", cfg.Path, err)
	}
	var creds credentials.Credentials
	var found bool
	if creds, found, err = stepCtx.CredentialsDB.Get(
		ctx,
		stepCtx.Project,
		credentials.TypeGit,
		workTree.URL(),
	); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error getting credentials for %s: %w", workTree.URL(), err)
	} else if found {
		loadOpts.Credentials = &git.RepoCredentials{
			Username:      creds.Username,
			Password:      creds.Password,
			SSHPrivateKey: creds.SSHPrivateKey,
		}
	}
	if workTree, err = git.LoadWorkTree(path, loadOpts); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error loading working tree from %s: %w", cfg.Path, err)
	}
	pushOpts := &git.PushOptions{
		// Start with whatever was specified in the config, which may be empty
		TargetBranch: cfg.TargetBranch,
	}
	// If we're supposed to generate a target branch name, do so
	if cfg.GenerateTargetBranch {
		pushOpts.TargetBranch = fmt.Sprintf("kargo/%s/%s/promotion", stepCtx.Project, stepCtx.Stage)
		pushOpts.Force = true
	}
	targetBranch := pushOpts.TargetBranch
	if targetBranch == "" {
		// If retBranch is still empty, we want to set it to the current branch
		// because we will want to return the branch that was pushed to, but we
		// don't want to mess with the options any further.
		if targetBranch, err = workTree.CurrentBranch(); err != nil {
			return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
				fmt.Errorf("error getting current branch: %w", err)
		}
	}
	if err = workTree.Push(pushOpts); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error pushing commits to remote: %w", err)
	}
	commitID, err := workTree.LastCommitID()
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error getting last commit ID: %w", err)
	}
	return PromotionStepResult{
		Status: kargoapi.PromotionPhaseSucceeded,
		Output: map[string]any{
			branchKey: targetBranch,
			commitKey: commitID,
		},
	}, nil
}
