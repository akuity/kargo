package directives

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/gitprovider"
)

func Test_gitPRWaiter_validate(t *testing.T) {
	testCases := []struct {
		name             string
		config           Config
		expectedProblems []string
	}{
		{
			name:   "repoURL not specified",
			config: Config{},
			expectedProblems: []string{
				"(root): repoURL is required",
			},
		},
		{
			name: "repoURL is empty string",
			config: Config{
				"repoURL": "",
			},
			expectedProblems: []string{
				"repoURL: String length must be greater than or equal to 1",
			},
		},
		{
			name:   "neither prNumber nor prNumberFromStep specified",
			config: Config{},
			expectedProblems: []string{
				"(root): Must validate one and only one schema",
			},
		},
		{
			name: "both prNumber and prNumberFromStep specified",
			config: Config{
				"prNumber":         42,
				"prNumberFromStep": "fake-step",
			},
			expectedProblems: []string{
				"(root): Must validate one and only one schema",
			},
		},
		{
			name: "provider is an invalid value",
			config: Config{
				"provider": "bogus",
			},
			expectedProblems: []string{
				"provider: provider must be one of the following:",
			},
		},
		{
			name: "valid without explicit provider",
			config: Config{
				"prNumber": 42,
				"repoURL":  "https://github.com/example/repo.git",
			},
		},
		{
			name: "valid with explicit provider",
			config: Config{
				"provider": "github",
				"prNumber": 42,
				"repoURL":  "https://github.com/example/repo.git",
			},
		},
	}

	r := newGitPRWaiter()
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

func Test_gitPRWaiter_runPromotionStep(t *testing.T) {
	testCases := []struct {
		name       string
		provider   gitprovider.GitProviderService
		assertions func(*testing.T, PromotionStepResult, error)
	}{
		{
			name: "error finding PR",
			provider: &gitprovider.FakeGitProviderService{
				GetPullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.ErrorContains(t, err, "error getting pull request")
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, PromotionStatusErrored, res.Status)
			},
		},
		{
			name: "PR is open",
			provider: &gitprovider.FakeGitProviderService{
				GetPullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, error) {
					return &gitprovider.PullRequest{
						State: gitprovider.PullRequestStateOpen,
					}, nil
				},
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, PromotionStatusRunning, res.Status)
			},
		},
		{
			name: "error checking if PR was merged",
			provider: &gitprovider.FakeGitProviderService{
				GetPullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, error) {
					return &gitprovider.PullRequest{
						State: gitprovider.PullRequestStateClosed,
					}, nil
				},
				IsPullRequestMergedFn: func(context.Context, int64) (bool, error) {
					return false, errors.New("something went wrong")
				},
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.ErrorContains(t, err, "error checking if pull request")
				require.ErrorContains(t, err, "was merged")
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, PromotionStatusErrored, res.Status)
			},
		},
		{
			name: "PR is closed and not merged",
			provider: &gitprovider.FakeGitProviderService{
				GetPullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, error) {
					return &gitprovider.PullRequest{
						State: gitprovider.PullRequestStateClosed,
					}, nil
				},
				IsPullRequestMergedFn: func(context.Context, int64) (bool, error) {
					return false, nil
				},
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.ErrorContains(t, err, "was closed without being merged")
				require.Equal(t, PromotionStatusFailed, res.Status)
			},
		},
		{
			name: "PR is closed and merged",
			provider: &gitprovider.FakeGitProviderService{
				GetPullRequestFn: func(
					context.Context,
					int64,
				) (*gitprovider.PullRequest, error) {
					return &gitprovider.PullRequest{
						State: gitprovider.PullRequestStateClosed,
					}, nil
				},
				IsPullRequestMergedFn: func(context.Context, int64) (bool, error) {
					return true, nil
				},
			},
			assertions: func(t *testing.T, res PromotionStepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, PromotionStatusSuccess, res.Status)
			},
		},
	}

	r := newGitPRWaiter()
	runner, ok := r.(*gitPRWaiter)
	require.True(t, ok)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Cannot register multiple providers with the same name, so this takes
			// care of that problem
			testGitProviderName := uuid.NewString()

			gitprovider.RegisterProvider(
				testGitProviderName,
				gitprovider.ProviderRegistration{
					NewService: func(
						string,
						*gitprovider.GitProviderOptions,
					) (gitprovider.GitProviderService, error) {
						return testCase.provider, nil
					},
				},
			)

			res, err := runner.runPromotionStep(
				context.Background(),
				&PromotionStepContext{
					CredentialsDB: &credentials.FakeDB{},
				},
				GitWaitForPRConfig{
					Provider: ptr.To(Provider(testGitProviderName)),
				},
			)
			testCase.assertions(t, res, err)
		})
	}
}
