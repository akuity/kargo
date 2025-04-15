package builtin

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/gitprovider"
	"github.com/akuity/kargo/pkg/promotion"
	promoPkg "github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_gitPRWaiter_validate(t *testing.T) {
	testCases := []struct {
		name             string
		config           promoPkg.Config
		expectedProblems []string
	}{
		{
			name:   "repoURL not specified",
			config: promoPkg.Config{},
			expectedProblems: []string{
				"(root): repoURL is required",
			},
		},
		{
			name: "repoURL is empty string",
			config: promoPkg.Config{
				"repoURL": "",
			},
			expectedProblems: []string{
				"repoURL: String length must be greater than or equal to 1",
			},
		},
		{
			name: "prNumber not specified",
			config: promoPkg.Config{
				"repoURL": "https://github.com/example/repo.git",
			},
			expectedProblems: []string{
				"(root): prNumber is required",
			},
		},
		{
			name: "prNumber is less than 1",
			config: promoPkg.Config{
				"prNumber": 0,
			},
			expectedProblems: []string{
				"prNumber: Must be greater than or equal to 1",
			},
		},
		{
			name: "provider is an invalid value",
			config: promoPkg.Config{
				"provider": "bogus",
			},
			expectedProblems: []string{
				"provider: provider must be one of the following:",
			},
		},
		{
			name: "valid without explicit provider",
			config: promoPkg.Config{
				"prNumber": 42,
				"repoURL":  "https://github.com/example/repo.git",
			},
		},
		{
			name: "valid with explicit provider",
			config: promoPkg.Config{
				"provider": "github",
				"prNumber": 42,
				"repoURL":  "https://github.com/example/repo.git",
			},
		},
	}

	r := newGitPRWaiter(nil)
	runner, ok := r.(*gitPRWaiter)
	require.True(t, ok)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := runner.validate(testCase.config)
			if len(testCase.expectedProblems) == 0 {
				require.NoError(t, err)
			} else {
				for _, problem := range testCase.expectedProblems {
					require.ErrorContains(t, err, problem)
				}
			}
		})
	}
}

func Test_gitPRWaiter_run(t *testing.T) {
	testCases := []struct {
		name       string
		provider   gitprovider.Interface
		assertions func(*testing.T, promoPkg.StepResult, error)
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
			assertions: func(t *testing.T, res promoPkg.StepResult, err error) {
				require.ErrorContains(t, err, "error getting pull request")
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, kargoapi.PromotionStepPhaseErrored, res.Status)
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
						Open: true,
					}, nil
				},
			},
			assertions: func(t *testing.T, res promoPkg.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepPhaseRunning, res.Status)
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
						Open:   false,
						Merged: false,
					}, nil
				},
			},
			assertions: func(t *testing.T, res promoPkg.StepResult, err error) {
				require.ErrorContains(t, err, "closed without being merged")
				require.True(t, promotion.IsTerminal(err))
				require.Equal(t, kargoapi.PromotionStepPhaseFailed, res.Status)
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
						Open:   false,
						Merged: true,
					}, nil
				},
			},
			assertions: func(t *testing.T, res promoPkg.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepPhaseSucceeded, res.Status)
			},
		},
	}

	r := newGitPRWaiter(&credentials.FakeDB{})
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
				&promoPkg.StepContext{},
				builtin.GitWaitForPRConfig{
					Provider: ptr.To(builtin.Provider(testGitProviderName)),
				},
			)
			testCase.assertions(t, res, err)
		})
	}
}
