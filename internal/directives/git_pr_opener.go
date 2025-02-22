package directives

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/gitprovider"

	_ "github.com/akuity/kargo/internal/gitprovider/azure"  // Azure provider registration
	_ "github.com/akuity/kargo/internal/gitprovider/gitea"  // Gitea provider registration
	_ "github.com/akuity/kargo/internal/gitprovider/github" // GitHub provider registration
	_ "github.com/akuity/kargo/internal/gitprovider/gitlab" // GitLab provider registration
)

// stateKeyPRNumber is the key used to store the PR number in the shared State.
const stateKeyPRNumber = "prNumber"

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
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, err
	}
	cfg, err := ConfigToStruct[GitOpenPRConfig](stepCtx.Config)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
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
	// Short-circuit if shared state has output from a previous execution of this
	// step that contains a PR number.
	prNumber, err := g.getPRNumber(stepCtx, stepCtx.SharedState)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error getting PR number from shared state: %w", err)
	}
	if prNumber != -1 {
		return PromotionStepResult{
			Status: kargoapi.PromotionPhaseSucceeded,
			Output: map[string]any{
				stateKeyPRNumber: prNumber,
			},
		}, nil
	}

	sourceBranch := cfg.SourceBranch

	var repoCreds *git.RepoCredentials
	creds, found, err := stepCtx.CredentialsDB.Get(
		ctx,
		stepCtx.Project,
		credentials.TypeGit,
		cfg.RepoURL,
	)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
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
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error cloning %s: %w", cfg.RepoURL, err)
	}
	defer repo.Close()

	gpOpts := &gitprovider.Options{
		InsecureSkipTLSVerify: cfg.InsecureSkipTLSVerify,
	}
	if repoCreds != nil {
		gpOpts.Token = repoCreds.Password
	}
	if cfg.Provider != nil {
		gpOpts.Name = string(*cfg.Provider)
	}
	gitProvider, err := gitprovider.New(cfg.RepoURL, gpOpts)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error creating git provider service: %w", err)
	}

	// If a PR somehow exists that is identical to the one we would open, we can
	// potentially just adopt it.
	pr, err := g.getExistingPR(
		ctx,
		repo,
		gitProvider,
		cfg.TargetBranch,
	)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error determining if pull request already exists: %w", err)
	}
	if pr != nil && (pr.Open || pr.Merged) { // Excludes PR that is both closed AND unmerged
		return PromotionStepResult{
			Status: kargoapi.PromotionPhaseSucceeded,
			Output: map[string]any{
				stateKeyPRNumber: pr.Number,
			},
		}, nil
	}

	// If we get to here, we either did not find an existing PR like the one we're
	// about to create, or we found one that is closed and not merged, which means
	// we're free to create a new one.

	// Get the title from the commit message of the head of the source branch
	// BEFORE we move on to ensuring the existence of the target branch because
	// that may involve creating a new branch and committing to it.
	commitMsg, err := repo.CommitMessage(sourceBranch)
	if err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, fmt.Errorf(
			"error getting commit message from head of branch %s: %w",
			sourceBranch, err,
		)
	}

	if err = g.ensureRemoteTargetBranch(
		repo,
		cfg.TargetBranch,
		cfg.CreateTargetBranch,
	); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, fmt.Errorf(
			"error ensuring existence of remote branch %s: %w",
			cfg.TargetBranch, err,
		)
	}

	var title string
	if cfg.Title != "" {
		title = cfg.Title
	} else {
		title = strings.Split(commitMsg, "\n")[0]
	}
	description := commitMsg
	if stepCtx.UIBaseURL != "" {
		description = fmt.Sprintf(
			"%s\n\n[View in Kargo UI](%s/project/%s/stage/%s)",
			description,
			stepCtx.UIBaseURL,
			stepCtx.Project,
			stepCtx.Stage,
		)
	}

	if pr, err = gitProvider.CreatePullRequest(
		ctx,
		&gitprovider.CreatePullRequestOpts{
			Head:        sourceBranch,
			Base:        cfg.TargetBranch,
			Title:       title,
			Description: description,
			Labels:      cfg.Labels,
		},
	); err != nil {
		return PromotionStepResult{Status: kargoapi.PromotionPhaseErrored},
			fmt.Errorf("error creating pull request: %w", err)
	}
	return PromotionStepResult{
		Status: kargoapi.PromotionPhaseSucceeded,
		Output: map[string]any{
			stateKeyPRNumber: pr.Number,
		},
	}, nil
}

// getPRNumber checks shared state for output from a previous execution of this
// step. If any is found and it contains a PR number, that number is returned.
// 0 is returned if no PR number is found in the shared state. An error is
// returned if the PR number is found but is neither an int64 nor a float64.
func (g *gitPROpener) getPRNumber(
	stepCtx *PromotionStepContext,
	sharedState State,
) (int64, error) {
	stepOutput, exists := sharedState.Get(stepCtx.Alias)
	if !exists {
		return -1, nil
	}
	stepOutputMap, ok := stepOutput.(map[string]any)
	if !ok {
		return -1, fmt.Errorf(
			"output from step with alias %q is not a map[string]any",
			stepCtx.Alias,
		)
	}
	prNumberAny, exists := stepOutputMap[stateKeyPRNumber]
	if !exists {
		return -1, nil
	}
	// If the state was rehydrated from PromotionStatus, which makes use of
	// apiextensions.JSON, the PR number will be a float64. Otherwise, it will be
	// an int64. We need to handle both cases.
	switch prNumber := prNumberAny.(type) {
	case int64:
		return prNumber, nil
	case float64:
		return int64(prNumber), nil
	default:
		return -1, fmt.Errorf(
			"PR number in output from step with alias %q is not an int64",
			stepCtx.Alias,
		)
	}
}

// ensureRemoteTargetBranch ensures the existence of a remote branch. If the
// branch does not exist, an empty orphaned branch is created and pushed to the
// remote.
func (g *gitPROpener) ensureRemoteTargetBranch(
	repo git.Repo,
	branch string, create bool,
) error {
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

// getExistingPR searches for an existing pull request from the head of the
// repo's current branch to the target branch. If a PR is found, it is returned.
// If no PR is found, nil is returned.
func (g *gitPROpener) getExistingPR(
	ctx context.Context,
	repo git.Repo,
	gitProv gitprovider.Interface,
	targetBranch string,
) (*gitprovider.PullRequest, error) {
	commitID, err := repo.LastCommitID()
	if err != nil {
		return nil, fmt.Errorf("error getting last commit ID: %w", err)
	}
	sourceBranch, err := repo.CurrentBranch()
	if err != nil {
		return nil, fmt.Errorf("error getting current branch: %w", err)
	}
	// Find any existing PRs that are identical to the one we might open.
	prs, err := gitProv.ListPullRequests(
		ctx,
		&gitprovider.ListPullRequestOptions{
			BaseBranch: targetBranch,
			HeadBranch: sourceBranch,
			HeadCommit: commitID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("error listing pull requests: %w", err)
	}
	if len(prs) == 0 {
		return nil, nil
	}
	// If promotion names are incorporated into PR source branches, it's highly
	// unlikely that we would have found more than one PR matching the search
	// criteria. Accounting for the possibility of users specifying their own
	// source branch names using an expression, although still unlikely, there is
	// somewhat more of a possibility of multiple PRs being found. In this case,
	// we need to determine which PR is best to "adopt" as a proxy for the PR we
	// would have otherwise opened. This requires sorting the PRs in a particular
	// order.
	g.sortPullRequests(prs)
	return &prs[0], nil
}

// sortPullRequests is a specialized sorting function that sorts pull requests
// in the following order: open PRs first, then closed PRs that have been
// merged, then closed PRs that have not been merged. Within each of those
// categories, PRs are sorted by creation time in descending order.
func (g *gitPROpener) sortPullRequests(prs []gitprovider.PullRequest) {
	slices.SortFunc(prs, func(lhs, rhs gitprovider.PullRequest) int {
		switch {
		case lhs.Open && !rhs.Open:
			// If the first PR is open and the second is not, the first PR should
			// come first.
			return -1
		case rhs.Open && !lhs.Open:
			// If the second PR is open and the first is not, the second PR should
			// come first.
			return 1
		case !lhs.Open && !rhs.Open:
			// If both PRs are closed, one is merged and one is not, the merged PR
			// should come first.
			if lhs.Merged && !rhs.Merged {
				return -1
			}
			if rhs.Merged && !lhs.Merged {
				return 1
			}
			// If we get to here, both PRs are closed and neither is merged. Fall
			// through to the default case.
			fallthrough
		default:
			// If we get to here, both PRs are open or both are closed and neither is
			// merged. The most recently opened PR should come first.
			var ltime time.Time
			if lhs.CreatedAt != nil {
				ltime = *lhs.CreatedAt
			}
			var rtime time.Time
			if rhs.CreatedAt != nil {
				rtime = *rhs.CreatedAt
			}
			return rtime.Compare(ltime)
		}
	})
}
