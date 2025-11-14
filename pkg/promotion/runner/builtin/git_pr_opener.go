package builtin

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/gitprovider"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"

	_ "github.com/akuity/kargo/pkg/gitprovider/azure"     // Azure provider registration
	_ "github.com/akuity/kargo/pkg/gitprovider/bitbucket" // Bitbucket provider registration
	_ "github.com/akuity/kargo/pkg/gitprovider/gitea"     // Gitea provider registration
	_ "github.com/akuity/kargo/pkg/gitprovider/github"    // GitHub provider registration
	_ "github.com/akuity/kargo/pkg/gitprovider/gitlab"    // GitLab provider registration
)

const stepKindGitOpenPR = "git-open-pr"

func init() {
	promotion.DefaultStepRunnerRegistry.MustRegister(
		promotion.StepRunnerRegistration{
			Name: stepKindGitOpenPR,
			Metadata: promotion.StepRunnerMetadata{
				RequiredCapabilities: []promotion.StepRunnerCapability{
					promotion.StepCapabilityAccessCredentials,
				},
			},
			Value: newGitPROpener,
		},
	)
}

// gitPROpener is an implementation of the promotion.StepRunner interface that
// opens a pull request.
type gitPROpener struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
}

// newGitPROpener returns an implementation of the promotion.StepRunner interface
// that opens a pull request.
func newGitPROpener(caps promotion.StepRunnerCapabilities) promotion.StepRunner {
	return &gitPROpener{
		credsDB:      caps.CredsDB,
		schemaLoader: getConfigSchemaLoader(stepKindGitOpenPR),
	}
}

// Run implements the promotion.StepRunner interface.
func (g *gitPROpener) Run(
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
// into a builtin.GitOpenPRConfig struct.
func (g *gitPROpener) convert(cfg promotion.Config) (builtin.GitOpenPRConfig, error) {
	return validateAndConvert[builtin.GitOpenPRConfig](g.schemaLoader, cfg, stepKindGitOpenPR)
}

func (g *gitPROpener) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.GitOpenPRConfig,
) (promotion.StepResult, error) {
	sourceBranch := cfg.SourceBranch

	// Short-circuit if shared state has output from a previous execution of this
	// step that contains a PR ID.
	prID, err := g.getPRID(stepCtx, stepCtx.SharedState)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error getting PR number from shared state: %w", err)
	}
	if prID != -1 {
		// We don't need to check for the existence of prior output from this
		// step because we'd not have gotten to here if it didn't exist.
		output, _ := stepCtx.SharedState.Get(stepCtx.Alias)
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusSucceeded,
			// Don't need to validate the type assertion on the output here because
			// we'd not have gotten to here if it wasn't a map[string]any.
			Output: output.(map[string]any), //nolint: forcetypeassert
		}, nil
	}

	var repoCreds *git.RepoCredentials
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
	if creds != nil {
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
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
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
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
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
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error determining if pull request already exists: %w", err)
	}

	if pr != nil && (pr.Open || pr.Merged) { // Excludes PR that is both closed AND unmerged
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusSucceeded,
			Output: map[string]any{
				"pr": map[string]any{
					"id":  pr.Number,
					"url": pr.URL,
				},
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
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, fmt.Errorf(
			"error getting commit message from head of branch %s: %w",
			sourceBranch, err,
		)
	}

	if err = g.ensureRemoteTargetBranch(
		repo,
		cfg.TargetBranch,
		cfg.CreateTargetBranch,
	); err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, fmt.Errorf(
			"error ensuring existence of remote branch %s: %w",
			cfg.TargetBranch, err,
		)
	}

	title := cfg.Title
	description := commitMsg

	if cfg.Description != "" {
		description = cfg.Description
	}

	if title == "" {
		parts := strings.SplitN(commitMsg, "\n", 2)
		title = parts[0]

		// Only override the description if it has not been set in the config.
		if cfg.Description == "" {
			if len(parts) > 1 {
				description = parts[1]
			} else {
				description = "" // The commit message is just a title.
			}
		}
	}

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
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error creating pull request: %w", err)
	}
	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: map[string]any{
			"pr": map[string]any{
				"id":  pr.Number,
				"url": pr.URL,
			},
		},
	}, nil
}

// getPRID checks shared state for output from a previous execution of this
// step. If any is found and it contains a PR ID, that ID is returned.
// -1 is returned if no PR ID is found in the shared state. An error is
// returned if the PR ID is found but is neither an int64 nor a float64.
func (g *gitPROpener) getPRID(
	stepCtx *promotion.StepContext,
	sharedState promotion.State,
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

	// Check for `pr.id`
	prMap, exists := stepOutputMap["pr"].(map[string]any)
	if !exists {
		return -1, nil
	}

	prIDAny, exists := prMap["id"]
	if !exists {
		return -1, nil
	}

	switch prID := prIDAny.(type) {
	case int64:
		return prID, nil
	case float64:
		return int64(prID), nil
	default:
		return -1, fmt.Errorf(
			"PR ID in output from step with alias %q is not an int64",
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
