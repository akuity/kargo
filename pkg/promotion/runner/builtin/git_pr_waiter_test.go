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

func Test_gitPRWaiter_convert(t *testing.T) {
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
	}

	r := newGitPRWaiter(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*gitPRWaiter)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_gitPRWaiter_run(t *testing.T) {
	testCases := []struct {
		name       string
		provider   gitprovider.Interface
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
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "error getting pull request")
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
			},
		},
		{
			name: "PR is open",
			provider: &gitprovider.Fake{
				GetPullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, error) {
					return &gitprovider.PullRequest{
						Number: 42,
						URL:    "https://github.com/example/repo/pull/42",
						Open:   true,
						Merged: false,
					}, nil
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusRunning, res.Status)
				// Verify PR metadata in output
				require.NotNil(t, res.Output)
				prOutput, ok := res.Output["pr"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, int64(42), prOutput["id"])
				require.Equal(t, "https://github.com/example/repo/pull/42", prOutput["url"])
				require.Equal(t, true, prOutput["open"])
				require.Equal(t, false, prOutput["merged"])
				// commit should not be present when PR is open
				_, hasCommit := res.Output["commit"]
				require.False(t, hasCommit)
			},
		},
		{
			name: "PR is closed and not merged",
			provider: &gitprovider.Fake{
				GetPullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, error) {
					return &gitprovider.PullRequest{
						Number: 42,
						URL:    "https://github.com/example/repo/pull/42",
						Open:   false,
						Merged: false,
					}, nil
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "closed without being merged")
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepStatusFailed, res.Status)
				// Verify PR metadata in output even on failure
				require.NotNil(t, res.Output)
				prOutput, ok := res.Output["pr"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, int64(42), prOutput["id"])
				require.Equal(t, "https://github.com/example/repo/pull/42", prOutput["url"])
				require.Equal(t, false, prOutput["open"])
				require.Equal(t, false, prOutput["merged"])
			},
		},
		{
			name: "PR is closed and merged",
			provider: &gitprovider.Fake{
				GetPullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, error) {
					return &gitprovider.PullRequest{
						Number:         42,
						URL:            "https://github.com/example/repo/pull/42",
						Open:           false,
						Merged:         true,
						MergeCommitSHA: "abc123def456",
					}, nil
				},
			},
			assertions: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				// Verify PR metadata in output
				require.NotNil(t, res.Output)
				prOutput, ok := res.Output["pr"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, int64(42), prOutput["id"])
				require.Equal(t, "https://github.com/example/repo/pull/42", prOutput["url"])
				require.Equal(t, false, prOutput["open"])
				require.Equal(t, true, prOutput["merged"])
				// Verify commit is present when merged
				require.Equal(t, "abc123def456", res.Output["commit"])
			},
		},
	}

	r := newGitPRWaiter(promotion.StepRunnerCapabilities{
		CredsDB: &credentials.FakeDB{},
	})
	runner, ok := r.(*gitPRWaiter)
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

			res, err := runner.run(
				context.Background(),
				&promotion.StepContext{},
				builtin.GitWaitForPRConfig{
					Provider: ptr.To(builtin.Provider(testGitProviderName)),
				},
			)
			testCase.assertions(t, res, err)
		})
	}
}
