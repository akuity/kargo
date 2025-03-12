package directives

import (
	"context"
	"fmt"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/internal/gitprovider"
	"github.com/akuity/kargo/pkg/x/directive/builtin"
)

func Test_gitPROpener_validate(t *testing.T) {
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
			name:   "targetBranch not specified",
			config: Config{},
			expectedProblems: []string{
				"(root): targetBranch is required",
			},
		},
		{
			name: "targetBranch is empty string",
			config: Config{
				"targetBranch": "",
			},
			expectedProblems: []string{
				"targetBranch: String length must be greater than or equal to 1",
			},
		},
		{
			name: "sourceBranch is empty string",
			config: Config{
				"sourceBranch": "",
			},
			expectedProblems: []string{
				"sourceBranch: String length must be greater than or equal to 1",
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
				"repoURL":      "https://github.com/example/repo.git",
				"sourceBranch": "fake-branch",
				"targetBranch": "another-fake-branch",
			},
		},
		{
			name: "valid with explicit provider",
			config: Config{
				"provider":     "github",
				"repoURL":      "https://github.com/example/repo.git",
				"sourceBranch": "fake-branch",
				"targetBranch": "another-fake-branch",
			},
		},
		{
			name: "valid with custom title",
			config: Config{
				"provider":     "github",
				"repoURL":      "https://github.com/example/repo.git",
				"sourceBranch": "fake-branch",
				"targetBranch": "another-fake-branch",
				"title":        "custom title",
			},
		},
	}

	p := newGitPROpener(nil)
	promoter, ok := p.(*gitPROpener)
	require.True(t, ok)

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := promoter.validate(testCase.config)
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

func Test_gitPROpener_promote(t *testing.T) {
	const testSourceBranch = "source"
	const testTargetBranch = "target"

	// Set up a test Git server in-process
	service := gitkit.New(
		gitkit.Config{
			Dir:        t.TempDir(),
			AutoCreate: true,
		},
	)
	require.NoError(t, service.Setup())
	server := httptest.NewServer(service)
	defer server.Close()

	// This is the URL of the "remote" repository
	testRepoURL := fmt.Sprintf("%s/test.git", server.URL)

	workDir := t.TempDir()

	repo, err := git.Clone(testRepoURL, nil, nil)
	require.NoError(t, err)
	defer repo.Close()
	err = repo.CreateOrphanedBranch(testSourceBranch)
	require.NoError(t, err)
	err = repo.Commit("Initial commit", &git.CommitOptions{AllowEmpty: true})
	require.NoError(t, err)
	err = repo.Push(nil)
	require.NoError(t, err)

	// Set up a fake git provider
	const fakeGitProviderName = "fake"
	const testPRNumber int64 = 42
	gitprovider.Register(
		fakeGitProviderName,
		gitprovider.Registration{
			NewProvider: func(
				string,
				*gitprovider.Options,
			) (gitprovider.Interface, error) {
				return &gitprovider.Fake{
					ListPullRequestsFn: func(
						context.Context,
						*gitprovider.ListPullRequestOptions,
					) ([]gitprovider.PullRequest, error) {
						// Avoid opening of a PR being short-circuited by simulating
						// conditions where the PR in question doesn't already exist.
						return nil, nil
					},
					CreatePullRequestFn: func(
						context.Context,
						*gitprovider.CreatePullRequestOpts,
					) (*gitprovider.PullRequest, error) {
						return &gitprovider.PullRequest{Number: testPRNumber}, nil
					},
				}, nil
			},
		},
	)

	// Now we can proceed to test gitPROpener...

	p := newGitPROpener(&credentials.FakeDB{})
	promoter, ok := p.(*gitPROpener)
	require.True(t, ok)

	res, err := promoter.promote(
		context.Background(),
		&PromotionStepContext{
			Project: "fake-project",
			Stage:   "fake-stage",
			WorkDir: workDir,
		},
		builtin.GitOpenPRConfig{
			RepoURL: testRepoURL,
			// We get slightly better coverage by using this option
			SourceBranch:       testSourceBranch,
			TargetBranch:       testTargetBranch,
			CreateTargetBranch: true,
			Provider:           ptr.To(builtin.Provider(fakeGitProviderName)),
			Title:              "kargo",
		},
	)
	require.NoError(t, err)
	prNumber, ok := res.Output[stateKeyPRNumber]
	require.True(t, ok)
	require.Equal(t, testPRNumber, prNumber)

	// Assert that the target branch, which didn't already exist, was created
	exists, err := repo.RemoteBranchExists(testTargetBranch)
	require.NoError(t, err)
	require.True(t, exists)
}

func Test_gitPROpener_sortPullRequests(t *testing.T) {
	newer := time.Now()
	older := newer.Add(-time.Hour)
	// These are laid out in the exact opposite order of how they should be
	// sorted. After sorting, we can assert the order is correct by comparing to
	// the reversed list.
	orig := []gitprovider.PullRequest{
		{
			Number:    6,
			Open:      false,
			Merged:    false,
			CreatedAt: &older,
		},
		{
			Number:    5,
			Open:      false,
			Merged:    false,
			CreatedAt: &newer,
		},
		{
			Number:    4,
			Open:      false,
			Merged:    true,
			CreatedAt: &older,
		},
		{
			Number:    3,
			Open:      false,
			Merged:    true,
			CreatedAt: &newer,
		},
		{
			Number:    2,
			Open:      true,
			Merged:    false,
			CreatedAt: &older,
		},
		{
			Number:    1,
			Open:      true,
			Merged:    false,
			CreatedAt: &newer,
		},
	}
	sorted := make([]gitprovider.PullRequest, len(orig))
	copy(sorted, orig)
	(&gitPROpener{}).sortPullRequests(sorted)
	slices.Reverse(orig)
	require.Equal(t, orig, sorted)
}
