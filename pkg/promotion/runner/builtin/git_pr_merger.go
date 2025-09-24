package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/gitprovider"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"

	_ "github.com/akuity/kargo/internal/gitprovider/azure"     // Azure provider registration
	_ "github.com/akuity/kargo/internal/gitprovider/bitbucket" // Bitbucket provider registration
	_ "github.com/akuity/kargo/internal/gitprovider/gitea"     // Gitea provider registration
	_ "github.com/akuity/kargo/internal/gitprovider/github"    // GitHub provider registration
	_ "github.com/akuity/kargo/internal/gitprovider/gitlab"    // GitLab provider registration
)

// gitPRMerger is an implementation of the promotion.StepRunner interface that
// merges a pull request.
type gitPRMerger struct {
	schemaLoader gojsonschema.JSONLoader
	credsDB      credentials.Database
}

// newGitPRMerger returns an implementation of the promotion.StepRunner interface
// that merges a pull request.
func newGitPRMerger(credsDB credentials.Database) promotion.StepRunner {
	r := &gitPRMerger{
		credsDB: credsDB,
	}
	r.schemaLoader = getConfigSchemaLoader(r.Name())
	return r
}

// Name implements the promotion.StepRunner interface.
func (g *gitPRMerger) Name() string {
	return "git-merge-pr"
}

// Run implements the promotion.StepRunner interface.
func (g *gitPRMerger) Run(
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
// into a builtin.GitMergePRConfig struct.
func (g *gitPRMerger) convert(cfg promotion.Config) (builtin.GitMergePRConfig, error) {
	return validateAndConvert[builtin.GitMergePRConfig](g.schemaLoader, cfg, g.Name())
}

func (g *gitPRMerger) run(
	ctx context.Context,
	stepCtx *promotion.StepContext,
	cfg builtin.GitMergePRConfig,
) (promotion.StepResult, error) {
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

	gpOpts := &gitprovider.Options{
		InsecureSkipTLSVerify: cfg.InsecureSkipTLSVerify,
	}
	if repoCreds != nil {
		gpOpts.Token = repoCreds.Password
	}
	if cfg.Provider != nil {
		gpOpts.Name = string(*cfg.Provider)
	}
	gitProv, err := gitprovider.New(cfg.RepoURL, gpOpts)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error creating git provider service: %w", err)
	}

	pr, err := gitProv.GetPullRequest(ctx, cfg.PRNumber)
	if err != nil {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored},
			fmt.Errorf("error getting pull request %d: %w", cfg.PRNumber, err)
	}

	// If PR is already merged, return success
	if pr.Merged {
		return promotion.StepResult{
			Status: kargoapi.PromotionStepStatusSucceeded,
			Output: map[string]any{stateKeyCommit: pr.MergeCommitSHA},
		}, nil
	}

	// If PR is closed but not merged, it's a terminal error
	if !pr.Open {
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{
				Err: fmt.Errorf("pull request %d is closed but not merged", cfg.PRNumber),
			}
	}

	// Try to merge the PR
	mergedPR, err := gitProv.MergePullRequest(ctx, cfg.PRNumber, &gitprovider.MergePullRequestOpts{
		CommitTitle:   fmt.Sprintf("Merge PR #%d", cfg.PRNumber),
		CommitMessage: "Automatically merged by Kargo promotion step",
	})
	if err != nil {
		// If waiting is enabled and the error suggests the PR is not ready to merge,
		// return RUNNING to retry later
		if cfg.Wait && g.isRetryableError(err) {
			return promotion.StepResult{Status: kargoapi.PromotionStepStatusRunning}, nil
		}
		return promotion.StepResult{Status: kargoapi.PromotionStepStatusFailed},
			&promotion.TerminalError{
				Err: fmt.Errorf("error merging pull request %d: %w", cfg.PRNumber, err),
			}
	}

	return promotion.StepResult{
		Status: kargoapi.PromotionStepStatusSucceeded,
		Output: map[string]any{stateKeyCommit: mergedPR.MergeCommitSHA},
	}, nil
}

// isRetryableError determines if an error indicates the PR is not ready to merge
// but might be ready later (e.g., waiting for checks, conflicts, etc.)
func (g *gitPRMerger) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Non-retryable error patterns (terminal failures) - check these FIRST
	nonRetryablePatterns := []string{
		"authentication failed",
		"unauthorized",
		"http 403", // Forbidden (usually permanent)
		"403 forbidden",
		"http 401", // Unauthorized
		"401 unauthorized",
		"http 404", // Not Found
		"404 not found",
		"http 400", // Bad Request (usually permanent)
		"400 bad request",
		"invalid token",
		"access denied",
		"permission denied",
		"insufficient permissions",
		"repository not found",
		"pull request not found",
		"merge request not found",
		"already merged",
		"already closed",
		"cannot reopen",
		"invalid request",
		"malformed",
		"bad request",
	}

	// If it matches a non-retryable pattern, definitely not retryable
	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(errStr, strings.ToLower(pattern)) {
			return false
		}
	}

	// GitHub-specific retryable error patterns
	githubRetryablePatterns := []string{
		"pull request is not mergeable",
		"mergeable_state",
		"head branch was modified",
		"base branch was modified",
		"required status checks",
		"at least one status check is expected",
		"pull request checks have not completed",
		"blocking status checks",
		"review required",
		"waiting for status to be reported",
		"pull request has review requested changes",
		"not in a mergeable state",
	}

	// GitLab-specific retryable error patterns
	gitlabRetryablePatterns := []string{
		"merge request cannot be merged",
		"merge request is not mergeable",
		"cannot merge",
		"pipeline must succeed",
		"is still running",
		"pipeline running",
		"all discussions must be resolved",
		"merge request must be approved",
		"work in progress",
		"draft",
		"merge blocked",
		"branch protection",
	}

	// Azure DevOps-specific retryable error patterns
	azureRetryablePatterns := []string{
		"pull request cannot be completed",
		"merge cannot be completed",
		"merge policies",
		"policy requirements",
		"required reviewers",
		"minimum number of reviewers",
		"pull request policy",
		"waiting for build",
		"build in progress",
		"work item linking policy",
		"comment requirements",
		"path length restriction",
	}

	// Gitea-specific retryable error patterns
	giteaRetryablePatterns := []string{
		"pull request is not mergeable",
		"merge conflict",
		"cannot be merged automatically",
		"pr is checking",
		"pull request must be mergeable",
		"required status check",
		"branch protection rule",
	}

	// Bitbucket-specific retryable error patterns
	bitbucketRetryablePatterns := []string{
		"pull request cannot be merged",
		"merge checks",
		"build status",
		"needs approval",
		"branch restrictions",
		"merge strategy",
		"unapproved changes",
	}

	// General HTTP status codes that often indicate retryable conditions
	httpRetryablePatterns := []string{
		"http 503", // Service Unavailable
		"503 service unavailable",
		"http 502", // Bad Gateway
		"502 bad gateway",
		"http 504", // Gateway Timeout
		"504 gateway timeout",
		"http 429", // Too Many Requests (rate limiting)
		"429 too many requests",
		"http 423", // Locked (resource temporarily locked)
		"423 locked",
		"http 422", // GitHub returns 422 for merge conflicts or validation errors
		"422 unprocessable entity",
		"http 409", // Conflict status often indicates temporary merge issues
		"409 conflict",
		"http 405", // Method not allowed - often temporary
		"405 method not allowed",
	}

	// General merge-related retryable patterns (cross-provider)
	generalRetryablePatterns := []string{
		"not mergeable",
		"cannot merge",
		"merge conflict",
		"head sha changed",
		"branch changed",
		"checks pending",
		"status pending",
		"build pending",
		"build running",
		"pipeline running",
		"checks running",
		"ci pending",
		"ci running",
		"approval required",
		"approval pending",
		"review required",
		"review pending",
		"reviewers",
		"changes requested",
		"discussions unresolved",
		"rebase needed",
		"behind base branch",
		"branch protection",
		"protected branch",
		"policy violation",
		"merge policies",
		"temporary",
		"try again",
		"retry",
		"not ready",
		"please wait",
		"in progress",
		"processing",
		"busy",
	}

	// Combine all retryable patterns
	allRetryablePatterns := append(githubRetryablePatterns, gitlabRetryablePatterns...)
	allRetryablePatterns = append(allRetryablePatterns, azureRetryablePatterns...)
	allRetryablePatterns = append(allRetryablePatterns, giteaRetryablePatterns...)
	allRetryablePatterns = append(allRetryablePatterns, bitbucketRetryablePatterns...)
	allRetryablePatterns = append(allRetryablePatterns, httpRetryablePatterns...)
	allRetryablePatterns = append(allRetryablePatterns, generalRetryablePatterns...)

	// Check if error message contains any retryable patterns
	for _, pattern := range allRetryablePatterns {
		if strings.Contains(errStr, strings.ToLower(pattern)) {
			return true
		}
	}

	// For any other errors, be conservative and don't retry
	// unless we explicitly know they're retryable
	return false
}
