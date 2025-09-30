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

	r := newGitPRMerger(promotion.StepRunnerCapabilities{
		CredsDB: &credentials.FakeDB{},
	})
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
			name: "error during merge attempt",
			provider: &gitprovider.Fake{
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
			name: "PR not ready to merge with wait enabled",
			provider: &gitprovider.Fake{
				MergePullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, bool, error) {
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
				require.Nil(t, res.Output)
			},
		},
		{
			name: "PR not ready to merge with wait disabled",
			provider: &gitprovider.Fake{
				MergePullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, bool, error) {
					return nil, false, nil
				},
			},
			config: builtin.GitMergePRConfig{
				PRNumber: 42,
				Wait:     false,
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "not ready to merge and wait is disabled")
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
			},
		},
		{
			name: "PR number validation",
			provider: &gitprovider.Fake{
				MergePullRequestFn: func(
					_ context.Context,
					prNumber int64,
				) (*gitprovider.PullRequest, bool, error) {
					require.Equal(t, int64(123), prNumber)
					return &gitprovider.PullRequest{
						MergeCommitSHA: "commit456",
					}, true, nil
				},
			},
			config: builtin.GitMergePRConfig{
				PRNumber: 123,
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Equal(t, "commit456", res.Output[stateKeyCommit])
			},
		},
		{
			name: "successful merge with empty merge commit SHA",
			provider: &gitprovider.Fake{
				MergePullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, bool, error) {
					return &gitprovider.PullRequest{
						MergeCommitSHA: "",
					}, true, nil
				},
			},
			config: builtin.GitMergePRConfig{
				PRNumber: 42,
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.Equal(t, "", res.Output[stateKeyCommit])
			},
		},
		{
			name: "successful PR merge",
			provider: &gitprovider.Fake{
				MergePullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, bool, error) {
					return &gitprovider.PullRequest{
						MergeCommitSHA: "abc123",
					}, true, nil
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
	}

	r := newGitPRMerger(promotion.StepRunnerCapabilities{
		CredsDB: &credentials.FakeDB{},
	})
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
