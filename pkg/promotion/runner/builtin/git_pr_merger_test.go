package builtin

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/gitprovider"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_gitPRMerger_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "repoURL not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): repoURL is required",
			},
		},
		{
			name: "repoURL is empty string",
			config: promotion.Config{
				"repoURL": "",
			},
			expectedProblems: []string{
				"repoURL: String length must be greater than or equal to 1",
			},
		},
		{
			name: "prNumber not specified",
			config: promotion.Config{
				"repoURL": "https://github.com/example/repo.git",
			},
			expectedProblems: []string{
				"(root): prNumber is required",
			},
		},
		{
			name: "prNumber is less than 1",
			config: promotion.Config{
				"prNumber": 0,
			},
			expectedProblems: []string{
				"prNumber: Must be greater than or equal to 1",
			},
		},
		{
			name: "provider is an invalid value",
			config: promotion.Config{
				"provider": "bogus",
			},
			expectedProblems: []string{
				"provider: provider must be one of the following:",
			},
		},
		{
			name: "valid without explicit provider",
			config: promotion.Config{
				"prNumber": 42,
				"repoURL":  "https://github.com/example/repo.git",
			},
		},
		{
			name: "valid with explicit provider",
			config: promotion.Config{
				"provider": "github",
				"prNumber": 42,
				"repoURL":  "https://github.com/example/repo.git",
			},
		},
		{
			name: "valid with wait enabled",
			config: promotion.Config{
				"provider": "github",
				"prNumber": 42,
				"repoURL":  "https://github.com/example/repo.git",
				"wait":     true,
			},
		},
	}

	r := newGitPRMerger(nil)
	runner, ok := r.(*gitPRMerger)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_gitPRMerger_run(t *testing.T) {
	testCases := []struct {
		name       string
		provider   gitprovider.Interface
		config     builtin.GitMergePRConfig
		assertions func(*testing.T, promotion.StepResult, error)
	}{
		{
			name: "error finding PR",
			provider: &gitprovider.Fake{
				GetPullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, error) {
					return nil, errors.New("something went wrong")
				},
			},
			config: builtin.GitMergePRConfig{
				PRNumber: 42,
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "error getting pull request")
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
			},
		},
		{
			name: "PR is already merged",
			provider: &gitprovider.Fake{
				GetPullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, error) {
					return &gitprovider.PullRequest{
						Open:           false,
						Merged:         true,
						MergeCommitSHA: "abc123",
					}, nil
				},
			},
			config: builtin.GitMergePRConfig{
				PRNumber: 42,
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Equal(t, "abc123", res.Output[stateKeyCommit])
			},
		},
		{
			name: "PR is closed but not merged",
			provider: &gitprovider.Fake{
				GetPullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, error) {
					return &gitprovider.PullRequest{
						Open:   false,
						Merged: false,
					}, nil
				},
			},
			config: builtin.GitMergePRConfig{
				PRNumber: 42,
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "closed but not merged")
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name: "PR is open but merge not implemented",
			provider: &gitprovider.Fake{
				GetPullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, error) {
					return &gitprovider.PullRequest{
						Open:   true,
						Merged: false,
					}, nil
				},
				MergePullRequestFn: func(
					context.Context,
					int64,
					*gitprovider.MergePullRequestOpts,
				) (*gitprovider.PullRequest, error) {
					return nil, errors.New("merge not implemented for this provider")
				},
			},
			config: builtin.GitMergePRConfig{
				PRNumber: 42,
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "error merging pull request")
				require.ErrorContains(t, err, "merge not implemented")
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name: "PR is open and successfully merged",
			provider: &gitprovider.Fake{
				GetPullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, error) {
					return &gitprovider.PullRequest{
						Open:   true,
						Merged: false,
					}, nil
				},
				MergePullRequestFn: func(
					context.Context,
					int64,
					*gitprovider.MergePullRequestOpts,
				) (*gitprovider.PullRequest, error) {
					return &gitprovider.PullRequest{
						Open:           false,
						Merged:         true,
						MergeCommitSHA: "merged123",
					}, nil
				},
			},
			config: builtin.GitMergePRConfig{
				PRNumber: 42,
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Equal(t, "merged123", res.Output[stateKeyCommit])
			},
		},
		{
			name: "PR is open but not mergeable - with wait enabled should return running",
			provider: &gitprovider.Fake{
				GetPullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, error) {
					return &gitprovider.PullRequest{
						Open:   true,
						Merged: false,
					}, nil
				},
				MergePullRequestFn: func(
					context.Context,
					int64,
					*gitprovider.MergePullRequestOpts,
				) (*gitprovider.PullRequest, error) {
					return nil, errors.New("PR is not mergeable - checks have not completed")
				},
			},
			config: builtin.GitMergePRConfig{
				PRNumber: 42,
				Wait:     true,
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
			},
		},
		{
			name: "PR merge with custom options",
			provider: &gitprovider.Fake{
				GetPullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, error) {
					return &gitprovider.PullRequest{
						Open:   true,
						Merged: false,
					}, nil
				},
				MergePullRequestFn: func(
					_ context.Context,
					prNumber int64,
					opts *gitprovider.MergePullRequestOpts,
				) (*gitprovider.PullRequest, error) {
					// Verify merge options
					require.Equal(t, int64(42), prNumber)
					require.NotNil(t, opts)
					require.Equal(t, "Merge PR #42", opts.CommitTitle)
					return &gitprovider.PullRequest{
						Open:           false,
						Merged:         true,
						MergeCommitSHA: "custom123",
					}, nil
				},
			},
			config: builtin.GitMergePRConfig{
				PRNumber: 42,
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Equal(t, "custom123", res.Output[stateKeyCommit])
			},
		},
	}

	r := newGitPRMerger(&credentials.FakeDB{})
	runner, ok := r.(*gitPRMerger)
	require.True(t, ok)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Cannot register multiple providers with the same name, so this takes
			// care of that problem
			testGitProviderName := uuid.NewString()

			gitprovider.Register(
				testGitProviderName,
				gitprovider.Registration{
					NewProvider: func(
						string,
						*gitprovider.Options,
					) (gitprovider.Interface, error) {
						return testCase.provider, nil
					},
				},
			)

			cfg := testCase.config
			cfg.Provider = ptr.To(builtin.Provider(testGitProviderName))
			cfg.RepoURL = "https://github.com/example/repo.git"

			res, err := runner.run(
				context.Background(),
				&promotion.StepContext{},
				cfg,
			)
			testCase.assertions(t, res, err)
		})
	}
}

func Test_gitPRMerger_isRetryableError(t *testing.T) {
	merger := &gitPRMerger{}

	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		// Nil error
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},

		// GitHub retryable errors
		{
			name:     "GitHub not mergeable",
			err:      errors.New("Pull request is not mergeable"),
			expected: true,
		},
		{
			name:     "GitHub mergeable state",
			err:      errors.New("mergeable_state is unknown"),
			expected: true,
		},
		{
			name:     "GitHub head branch modified",
			err:      errors.New("head branch was modified. Review and try again"),
			expected: true,
		},
		{
			name:     "GitHub required status checks",
			err:      errors.New("Required status checks are pending"),
			expected: true,
		},
		{
			name:     "GitHub checks not completed",
			err:      errors.New("Pull request checks have not completed"),
			expected: true,
		},
		{
			name:     "GitHub review required",
			err:      errors.New("Review required before merging"),
			expected: true,
		},
		{
			name:     "GitHub 422 validation error",
			err:      errors.New("HTTP 422: Validation Failed"),
			expected: true,
		},
		{
			name:     "GitHub 409 conflict",
			err:      errors.New("HTTP 409: Conflict"),
			expected: true,
		},

		// GitLab retryable errors
		{
			name:     "GitLab cannot merge",
			err:      errors.New("Merge request cannot be merged"),
			expected: true,
		},
		{
			name:     "GitLab pipeline must succeed",
			err:      errors.New("Pipeline must succeed. It's currently pending"),
			expected: true,
		},
		{
			name:     "GitLab discussions unresolved",
			err:      errors.New("All discussions must be resolved"),
			expected: true,
		},
		{
			name:     "GitLab approval required",
			err:      errors.New("Merge request must be approved"),
			expected: true,
		},
		{
			name:     "GitLab work in progress",
			err:      errors.New("Cannot merge work in progress merge request"),
			expected: true,
		},
		{
			name:     "GitLab draft MR",
			err:      errors.New("Cannot merge draft merge request"),
			expected: true,
		},

		// Azure DevOps retryable errors
		{
			name:     "Azure cannot complete",
			err:      errors.New("Pull request cannot be completed due to policy requirements"),
			expected: true,
		},
		{
			name:     "Azure merge policies",
			err:      errors.New("Merge policies are not satisfied"),
			expected: true,
		},
		{
			name:     "Azure required reviewers",
			err:      errors.New("Required reviewers policy not met"),
			expected: true,
		},
		{
			name:     "Azure build in progress",
			err:      errors.New("Waiting for build to complete"),
			expected: true,
		},

		// Gitea retryable errors
		{
			name:     "Gitea not mergeable",
			err:      errors.New("Pull request is not mergeable"),
			expected: true,
		},
		{
			name:     "Gitea merge conflict",
			err:      errors.New("Merge conflict detected"),
			expected: true,
		},
		{
			name:     "Gitea cannot merge automatically",
			err:      errors.New("Cannot be merged automatically"),
			expected: true,
		},
		{
			name:     "Gitea checking",
			err:      errors.New("PR is checking, please wait"),
			expected: true,
		},

		// Bitbucket retryable errors
		{
			name:     "Bitbucket merge checks",
			err:      errors.New("Pull request merge checks are pending"),
			expected: true,
		},
		{
			name:     "Bitbucket build status",
			err:      errors.New("Build status must pass before merging"),
			expected: true,
		},
		{
			name:     "Bitbucket needs approval",
			err:      errors.New("Pull request needs approval before merging"),
			expected: true,
		},

		// HTTP status code retryable errors
		{
			name:     "HTTP 503 Service Unavailable",
			err:      errors.New("HTTP 503: Service Unavailable"),
			expected: true,
		},
		{
			name:     "HTTP 502 Bad Gateway",
			err:      errors.New("HTTP 502: Bad Gateway"),
			expected: true,
		},
		{
			name:     "HTTP 504 Gateway Timeout",
			err:      errors.New("HTTP 504: Gateway Timeout"),
			expected: true,
		},
		{
			name:     "HTTP 429 Rate Limited",
			err:      errors.New("HTTP 429: Too Many Requests - Rate limit exceeded"),
			expected: true,
		},
		{
			name:     "HTTP 423 Locked",
			err:      errors.New("HTTP 423: Resource temporarily locked"),
			expected: true,
		},

		// General retryable patterns
		{
			name:     "General not mergeable",
			err:      errors.New("Branch not mergeable due to conflicts"),
			expected: true,
		},
		{
			name:     "General checks pending",
			err:      errors.New("Checks pending completion"),
			expected: true,
		},
		{
			name:     "General build running",
			err:      errors.New("Build running, please wait"),
			expected: true,
		},
		{
			name:     "General pipeline running",
			err:      errors.New("Pipeline running"),
			expected: true,
		},
		{
			name:     "General approval pending",
			err:      errors.New("Approval pending from reviewers"),
			expected: true,
		},
		{
			name:     "General behind base branch",
			err:      errors.New("Branch is behind base branch"),
			expected: true,
		},
		{
			name:     "General branch protection",
			err:      errors.New("Branch protection rules prevent merge"),
			expected: true,
		},
		{
			name:     "General rebase needed",
			err:      errors.New("Rebase needed before merge"),
			expected: true,
		},
		{
			name:     "General temporary error",
			err:      errors.New("Temporary merge failure, please try again"),
			expected: true,
		},

		// Non-retryable errors (terminal failures)
		{
			name:     "Authentication failed",
			err:      errors.New("Authentication failed - invalid token"),
			expected: false,
		},
		{
			name:     "HTTP 401 Unauthorized",
			err:      errors.New("HTTP 401: Unauthorized"),
			expected: false,
		},
		{
			name:     "HTTP 403 Forbidden",
			err:      errors.New("HTTP 403: Forbidden - insufficient permissions"),
			expected: false,
		},
		{
			name:     "HTTP 404 Not Found",
			err:      errors.New("HTTP 404: Pull request not found"),
			expected: false,
		},
		{
			name:     "HTTP 400 Bad Request",
			err:      errors.New("HTTP 400: Bad Request - malformed request"),
			expected: false,
		},
		{
			name:     "Access denied",
			err:      errors.New("Access denied to repository"),
			expected: false,
		},
		{
			name:     "Repository not found",
			err:      errors.New("Repository not found or access denied"),
			expected: false,
		},
		{
			name:     "Pull request not found",
			err:      errors.New("Pull request #123 not found"),
			expected: false,
		},
		{
			name:     "Already merged",
			err:      errors.New("Pull request is already merged"),
			expected: false,
		},
		{
			name:     "Already closed",
			err:      errors.New("Pull request is already closed"),
			expected: false,
		},
		{
			name:     "Invalid request",
			err:      errors.New("Invalid request parameters"),
			expected: false,
		},
		{
			name:     "Malformed request",
			err:      errors.New("Malformed JSON in request body"),
			expected: false,
		},
		{
			name:     "Permission denied",
			err:      errors.New("Permission denied for this operation"),
			expected: false,
		},
		{
			name:     "Invalid token",
			err:      errors.New("Invalid or expired authentication token"),
			expected: false,
		},

		// Edge cases
		{
			name:     "Empty error message",
			err:      errors.New(""),
			expected: false,
		},
		{
			name:     "Mixed case patterns",
			err:      errors.New("Pull Request Is Not MERGEABLE due to conflicts"),
			expected: true,
		},
		{
			name: "Complex error with retryable pattern",
			err: errors.New("API Error: The pull request cannot be merged because " +
				"required status checks are still pending completion. Please wait for all checks to finish"),
			expected: true,
		},
		{
			name: "Complex error with non-retryable pattern",
			err: errors.New("API Error: Pull request #456 not found in repository owner/repo. " +
				"Verify the pull request number and repository access"),
			expected: false,
		},
		{
			name:     "Network-style error",
			err:      errors.New("dial tcp: i/o timeout"),
			expected: false, // Network errors are typically handled by higher-level retry logic
		},
		{
			name:     "Context canceled",
			err:      errors.New("context canceled"),
			expected: false, // Context cancellation is not retryable in this context
		},

		// Case sensitivity tests
		{
			name:     "Upper case merge conflict",
			err:      errors.New("MERGE CONFLICT DETECTED"),
			expected: true,
		},
		{
			name:     "Mixed case authentication",
			err:      errors.New("Authentication Failed"),
			expected: false,
		},

		// Real-world API response patterns
		{
			name: "GitHub API 422 response with detailed message",
			err: errors.New("PUT https://api.github.com/repos/owner/repo/pulls/123/merge: " +
				"422 Pull request is not mergeable due to failing required status checks"),
			expected: true,
		},
		{
			name:     "GitLab API 405 pipeline pending",
			err:      errors.New("405 (Method Not Allowed): Pipeline #12345 is still running"),
			expected: true,
		},
		{
			name: "Azure DevOps policy error",
			err: errors.New("TF400898: An Internal Error Occurred. Activity Id: abc123. " +
				"Required pull request policy 'Minimum number of reviewers' has not been satisfied"),
			expected: true,
		},
		{
			name:     "Gitea API error with checking status",
			err:      errors.New("422 Unprocessable Entity: PR is checking, please wait for the CI to complete"),
			expected: true,
		},
		{
			name:     "Bitbucket insufficient approvals",
			err:      errors.New("Pull request #123 cannot be merged: Insufficient branch permissions or unapproved changes"),
			expected: true,
		},
		{
			name: "GitHub rate limit with retry hint",
			err: errors.New("403 API rate limit exceeded for user ID 12345. " +
				"(But here's the good news: Authenticated requests get a higher rate limit. " +
				"Check out the documentation for more details.)"),
			expected: false, // 403 is non-retryable due to first check
		},
		{
			name: "Complex Jenkins/CI integration error",
			err: errors.New("merge blocked: Required status check 'continuous-integration/jenkins/pr-merge' " +
				"is pending. The PR will be ready for merge once all required checks pass"),
			expected: true,
		},
		{
			name: "Branch protection with specific rule",
			err: errors.New("Cannot merge: 2 of 3 required status checks are still pending: " +
				"'ci/circleci: test', 'ci/circleci: lint'"),
			expected: true,
		},
		{
			name: "GitLab WIP merge request",
			err: errors.New("405 Method Not Allowed: Cannot accept this merge request, " +
				"it is a Work in Progress. Remove the WIP prefix from the title to allow it to be merged"),
			expected: true,
		},
		{
			name: "Azure build validation failure",
			err: errors.New("The pull request cannot be completed because one or more policies are not satisfied: " +
				"Build validation policy requires a successful build before completing the pull request"),
			expected: true,
		},

		// Performance and edge cases
		{
			name: "Very long error message with retryable pattern",
			err: errors.New("This is a very long error message that contains many words and describes " +
				"a complex scenario where the pull request cannot be merged due to various policy " +
				"requirements and the fact that required status checks are still pending completion " +
				"from the continuous integration system which is currently processing the latest " +
				"changes pushed to the source branch and will need additional time to complete " +
				"its validation processes"),
			expected: true,
		},
		{
			name: "JSON formatted error response",
			err: errors.New(`{"message":"Pull request is not mergeable",` +
				`"documentation_url":"https://docs.github.com/rest/reference/pulls#merge-a-pull-request"}`),
			expected: true,
		},
		{
			name: "URL with error in query params",
			err: errors.New("GET https://gitlab.example.com/api/v4/projects/123/merge_requests/456/merge" +
				"?merge_when_pipeline_succeeds=true returned 409: Merge request cannot be merged"),
			expected: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := merger.isRetryableError(testCase.err)
			if testCase.err == nil {
				require.Equal(t, testCase.expected, result,
					"Expected isRetryableError(nil) to return %v, but got %v",
					testCase.expected, result)
			} else {
				require.Equal(t, testCase.expected, result,
					"Expected isRetryableError(%q) to return %v, but got %v",
					testCase.err.Error(), testCase.expected, result)
			}
		})
	}
}
