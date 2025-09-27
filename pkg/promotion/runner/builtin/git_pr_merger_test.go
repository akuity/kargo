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
			name: "error during merge attempt",
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
				) (*gitprovider.PullRequest, bool, error) {
					return nil, false, errors.New("authentication failed")
				},
			},
			config: builtin.GitMergePRConfig{
				PRNumber: 42,
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "error merging pull request")
				require.ErrorContains(t, err, "authentication failed")
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name: "PR is not ready to merge - with wait enabled should return running",
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
				) (*gitprovider.PullRequest, bool, error) {
					// Not ready to merge (checks pending), but no error
					return nil, false, nil
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
			name: "PR is not ready to merge - with wait disabled should fail",
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
				) (*gitprovider.PullRequest, bool, error) {
					// Not ready to merge (checks pending), but no error
					return nil, false, nil
				},
			},
			config: builtin.GitMergePRConfig{
				PRNumber: 42,
				Wait:     false, // or omit since false is default
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "not ready to merge and wait is disabled")
				require.True(t, promotion.IsTerminal(err))
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
				) (*gitprovider.PullRequest, bool, error) {
					return &gitprovider.PullRequest{
						Open:           false,
						Merged:         true,
						MergeCommitSHA: "merged123",
					}, true, nil
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
			name: "PR merge uses provider defaults",
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
				) (*gitprovider.PullRequest, bool, error) {
					// Verify PR number
					require.Equal(t, int64(42), prNumber)
					return &gitprovider.PullRequest{
						Open:           false,
						Merged:         true,
						MergeCommitSHA: "default123",
					}, true, nil
				},
			},
			config: builtin.GitMergePRConfig{
				PRNumber: 42,
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Equal(t, "default123", res.Output[stateKeyCommit])
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
