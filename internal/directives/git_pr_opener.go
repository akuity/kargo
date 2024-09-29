package directives

import (
	"context"
	"fmt"

	"github.com/xeipuuv/gojsonschema"

	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/gitprovider"
)

const prNumberKey = "prNumber"

func init() {
	builtins.RegisterPromotionStepRunner(
		newGitPROpener(),
		&StepRunnerPermissions{AllowCredentialsDB: true},
	)
}

// gitPROpener is an implementation of the PromotionStepRunner interface that
// opens a pull request.
type gitPROpener struct {
	schemaLoader gojsonschema.JSONLoader
}

// newGitPROpener returns an implementation of the PromotionStepRunner interface
// that opens a pull request.
func newGitPROpener() PromotionStepRunner {
	r := &gitPROpener{}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the PromotionStepRunner interface.
func (g *gitPROpener) Name() string {
	return "git-open-pr"
}

// RunPromotionStep implements the PromotionStepRunner interface.
func (g *gitPROpener) RunPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
) (PromotionStepResult, error) {
	if err := g.validate(stepCtx.Config); err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, err
	}
	cfg, err := configToStruct[GitOpenPRConfig](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("could not convert config into git-open-pr config: %w", err)
	}
	return g.runPromotionStep(ctx, stepCtx, cfg)
}

// validate validates gitPROpener configuration against a JSON schema.
func (g *gitPROpener) validate(cfg Config) error {
	return validate(g.schemaLoader, gojsonschema.NewGoLoader(cfg), g.Name())
}

func (g *gitPROpener) runPromotionStep(
	ctx context.Context,
	stepCtx *PromotionStepContext,
	cfg GitOpenPRConfig,
) (PromotionStepResult, error) {
	sourceBranch, err := getSourceBranch(stepCtx.SharedState, cfg)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error determining source branch: %w", err)
	}

	var repoCreds *git.RepoCredentials
	creds, found, err := stepCtx.CredentialsDB.Get(
		ctx,
		stepCtx.Project,
		credentials.TypeGit,
		cfg.RepoURL,
	)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error getting credentials for %s: %w", cfg.RepoURL, err)
	}
	if found {
		repoCreds = &git.RepoCredentials{
			Username:      creds.Username,
			Password:      creds.Password,
			SSHPrivateKey: creds.SSHPrivateKey,
		}
	}

	repo, err := git.Clone(
		cfg.RepoURL,
		&git.ClientOptions{
			Credentials:           repoCreds,
			InsecureSkipTLSVerify: cfg.InsecureSkipTLSVerify,
		},
		&git.CloneOptions{
			Depth:  1,
			Branch: sourceBranch,
		},
	)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error cloning %s: %w", cfg.RepoURL, err)
	}
	defer repo.Close()

	gpOpts := &gitprovider.GitProviderOptions{
		InsecureSkipTLSVerify: cfg.InsecureSkipTLSVerify,
	}
	if repoCreds != nil {
		gpOpts.Token = repoCreds.Password
	}
	if cfg.Provider != nil {
		gpOpts.Name = string(*cfg.Provider)
	}
	gitProviderSvc, err := gitprovider.NewGitProviderService(cfg.RepoURL, gpOpts)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error creating git provider service: %w", err)
	}

	// Short-circuit if a pull request already exists with the same head commit
	mustOpen, err := mustOpenPR(
		ctx,
		repo,
		gitProviderSvc,
		sourceBranch,
		cfg.TargetBranch,
	)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error determining if pull request must be opened: %w", err)
	}
	if !mustOpen {
		return PromotionStepResult{Status: PromotionStatusSuccess}, nil
	}

	// Get the title from the commit message of the head of the source branch
	// BEFORE we move on to ensuring the existence of the target branch because
	// that may involve creating a new branch and committing to it.
	title, err := repo.CommitMessage(sourceBranch)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, fmt.Errorf(
			"error getting commit message from head of branch %s: %w",
			sourceBranch, err,
		)
	}

	if err = ensureRemoteTargetBranch(
		repo,
		cfg.TargetBranch,
		cfg.CreateTargetBranch,
	); err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure}, fmt.Errorf(
			"error ensuring existence of remote branch %s: %w",
			cfg.TargetBranch, err,
		)
	}

	pr, err := gitProviderSvc.CreatePullRequest(
		ctx,
		gitprovider.CreatePullRequestOpts{
			Head:  sourceBranch,
			Base:  cfg.TargetBranch,
			Title: title,
		},
	)
	if err != nil {
		return PromotionStepResult{Status: PromotionStatusFailure},
			fmt.Errorf("error creating pull request: %w", err)
	}
	return PromotionStepResult{
		Status: PromotionStatusSuccess,
		Output: map[string]any{
			prNumberKey: pr.Number,
		},
	}, nil
}

func getSourceBranch(sharedState State, cfg GitOpenPRConfig) (string, error) {
	sourceBranch := cfg.SourceBranch
	if cfg.SourceBranchFromStep != "" {
		stepOutput, exists := sharedState.Get(cfg.SourceBranchFromStep)
		if !exists {
			return "", fmt.Errorf(
				"no output found from step with alias %q",
				cfg.SourceBranchFromStep,
			)
		}
		stepOutputMap, ok := stepOutput.(map[string]any)
		if !ok {
			return "", fmt.Errorf(
				"output from step with alias %q is not a mao[string]any",
				cfg.SourceBranchFromStep,
			)
		}
		sourceBranchAny, exists := stepOutputMap[branchKey]
		if !exists {
			return "", fmt.Errorf(
				"no branch found in output from step with alias %q",
				cfg.SourceBranchFromStep,
			)
		}
		if sourceBranch, ok = sourceBranchAny.(string); !ok {
			return "", fmt.Errorf(
				"branch name in output from step with alias %q is not a string",
				cfg.SourceBranchFromStep,
			)
		}
	}
	return sourceBranch, nil
}

// ensureRemoteTargetBranch ensures the existence of a remote branch. If the
// branch does not exist, an empty orphaned branch is created and pushed to the
// remote.
func ensureRemoteTargetBranch(repo git.Repo, branch string, create bool) error {
	exists, err := repo.RemoteBranchExists(branch)
	if err != nil {
		return fmt.Errorf(
			"error checking if remote branch %q of repo %s exists: %w",
			branch, repo.URL(), err,
		)
	}
	if exists {
		return nil
	}
	if !create {
		return fmt.Errorf(
			"remote branch %q does not exist in repo %s", branch, repo.URL(),
		)
	}
	if err = repo.CreateOrphanedBranch(branch); err != nil {
		return fmt.Errorf(
			"error creating orphaned branch %q in repo %s: %w",
			branch, repo.URL(), err,
		)
	}
	if err = repo.Commit(
		"Initial commit",
		&git.CommitOptions{AllowEmpty: true},
	); err != nil {
		return fmt.Errorf(
			"error making initial commit to new branch %q of repo %s: %w",
			branch, repo.URL(), err,
		)
	}
	if err = repo.Push(&git.PushOptions{TargetBranch: branch}); err != nil {
		return fmt.Errorf(
			"error pushing initial commit to new branch %q to repo %s: %w",
			branch, repo.URL(), err,
		)
	}
	return nil
}

// mustOpenPR determines if a pull request must be opened. It returns true if no
// PR exists for the head commit of the source branch to the target branch.
// Whether the PR is open or closed is irrelevant as we must NOT create a new PR
// if one already exists for the same head commit and has already been closed.
func mustOpenPR(
	ctx context.Context,
	repo git.Repo,
	gitProviderSvc gitprovider.GitProviderService,
	sourceBranch,
	targetBranch string,
) (bool, error) {
	commitID, err := repo.LastCommitID()
	if err != nil {
		return false, fmt.Errorf("error getting last commit ID: %w", err)
	}
	prs, err := gitProviderSvc.ListPullRequests(
		ctx,
		gitprovider.ListPullRequestOpts{
			Base: targetBranch,
			Head: sourceBranch,
		},
	)
	if err != nil {
		return false, fmt.Errorf("error listing pull requests: %w", err)
	}
	if len(prs) == 0 {
		return true, nil
	}
	for _, pr := range prs {
		if pr.HeadSHA == commitID {
			return false, nil
		}
	}
	return true, nil
}
