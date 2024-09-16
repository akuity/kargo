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

func TestGitWaitForPRDirective_Validate(t *testing.T) {
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
			name:   "neither prNumber nor prNumberFromOpen specified",
			config: Config{},
			expectedProblems: []string{
				"(root): Must validate one and only one schema",
			},
		},
		{
			name: "both prNumber and prNumberFromOpen specified",
			config: Config{
				"prNumber":         42,
				"prNumberFromOpen": "fake-step",
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

	d := newGitWaitForPRDirective()
	dir, ok := d.(*gitWaitForPRDirective)
	require.True(t, ok)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := dir.validate(testCase.config)
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

func TestGitWaitForPRDirective__Run(t *testing.T) {
	testCases := []struct {
		name       string
		provider   gitprovider.GitProviderService
		assertions func(*testing.T, Result, error)
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
			assertions: func(t *testing.T, res Result, err error) {
				require.ErrorContains(t, err, "error getting pull request")
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, StatusFailure, res.Status)
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
			assertions: func(t *testing.T, res Result, err error) {
				require.NoError(t, err)
				require.Equal(t, StatusPending, res.Status)
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
			assertions: func(t *testing.T, res Result, err error) {
				require.ErrorContains(t, err, "error checking if pull request")
				require.ErrorContains(t, err, "was merged")
				require.ErrorContains(t, err, "something went wrong")
				require.Equal(t, StatusFailure, res.Status)
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
			assertions: func(t *testing.T, res Result, err error) {
				require.ErrorContains(t, err, "was closed without being merged")
				require.Equal(t, StatusFailure, res.Status)
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
			assertions: func(t *testing.T, res Result, err error) {
				require.NoError(t, err)
				require.Equal(t, StatusSuccess, res.Status)
			},
		},
	}

	d := newGitWaitForPRDirective()
	dir, ok := d.(*gitWaitForPRDirective)
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

			res, err := dir.run(
				context.Background(),
				&StepContext{
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
