package builtin

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/credentials"
	"github.com/akuity/kargo/pkg/gitprovider"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_gitPROpener_convert(t *testing.T) {
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
			name:   "targetBranch not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): targetBranch is required",
			},
		},
		{
			name: "targetBranch is empty string",
			config: promotion.Config{
				"targetBranch": "",
			},
			expectedProblems: []string{
				"targetBranch: String length must be greater than or equal to 1",
			},
		},
		{
			name: "sourceBranch is empty string",
			config: promotion.Config{
				"sourceBranch": "",
			},
			expectedProblems: []string{
				"sourceBranch: String length must be greater than or equal to 1",
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
				"repoURL":      "https://github.com/example/repo.git",
				"sourceBranch": "fake-branch",
				"targetBranch": "another-fake-branch",
			},
		},
		{
			name: "valid with explicit provider",
			config: promotion.Config{
				"provider":     "github",
				"repoURL":      "https://github.com/example/repo.git",
				"sourceBranch": "fake-branch",
				"targetBranch": "another-fake-branch",
			},
		},
		{
			name: "valid with custom title",
			config: promotion.Config{
				"provider":     "github",
				"repoURL":      "https://github.com/example/repo.git",
				"sourceBranch": "fake-branch",
				"targetBranch": "another-fake-branch",
				"title":        "custom title",
			},
		},
		{
			name: "invalid with empty title",
			config: promotion.Config{
				"provider":     "github",
				"repoURL":      "https://github.com/example/repo.git",
				"sourceBranch": "fake-branch",
				"targetBranch": "another-fake-branch",
				"title":        "",
			},
			expectedProblems: []string{
				"title: String length must be greater than or equal to 1",
			},
		},
		{
			name: "valid with custom description",
			config: promotion.Config{
				"provider":     "github",
				"repoURL":      "https://github.com/example/repo.git",
				"sourceBranch": "fake-branch",
				"targetBranch": "another-fake-branch",
				"description":  "custom description",
			},
		},
		{
			name: "invalid with empty description",
			config: promotion.Config{
				"provider":     "github",
				"repoURL":      "https://github.com/example/repo.git",
				"sourceBranch": "fake-branch",
				"targetBranch": "another-fake-branch",
				"description":  "",
			},
			expectedProblems: []string{
				"description: String length must be greater than or equal to 1",
			},
		},
	}

	r := newGitPROpener(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*gitPROpener)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_gitPROpener_run(t *testing.T) {
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

	// Make a file with the same name as the branch
	err = os.WriteFile(filepath.Join(repo.Dir(), testSourceBranch), []byte("foo"), 0600)
	require.NoError(t, err)

	err = repo.AddAll()
	require.NoError(t, err)

	err = repo.Commit("Initial commit", &git.CommitOptions{AllowEmpty: true})
	require.NoError(t, err)
	err = repo.Push(nil)
	require.NoError(t, err)

	// Set up a fake git provider
	const fakeGitProviderName = "fake"
	const testPRNumber int64 = 42
	const testPRURL = "http://example.com/pull/42"
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
						return &gitprovider.PullRequest{
							Number: testPRNumber,
							URL:    testPRURL,
						}, nil
					},
				}, nil
			},
		},
	)

	// Now we can proceed to test gitPROpener...

	t.Run("fail when remote target branch does not exist", func(t *testing.T) {
		r := newGitPROpener(promotion.StepRunnerCapabilities{
			CredsDB: &credentials.FakeDB{},
		})
		runner, ok := r.(*gitPROpener)
		require.True(t, ok)

		res, err := runner.run(
			t.Context(),
			&promotion.StepContext{
				Project: "fake-project",
				Stage:   "fake-stage",
				WorkDir: workDir,
			},
			builtin.GitOpenPRConfig{
				RepoURL:      testRepoURL,
				SourceBranch: testSourceBranch,
				TargetBranch: testTargetBranch,
				Provider:     ptr.To(builtin.Provider(fakeGitProviderName)),
			},
		)
		require.Error(t, err)
		require.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
		require.ErrorContains(t, err, "target branch \"target\" does not exist")
	})

	t.Run("skip when target branch exists but has no changes", func(t *testing.T) {
		// push the target branch with no changes compared to the source branch
		require.NoError(t, repo.CreateChildBranch(testTargetBranch))
		require.NoError(t, repo.Push(nil))

		r := newGitPROpener(promotion.StepRunnerCapabilities{
			CredsDB: &credentials.FakeDB{},
		})
		runner, ok := r.(*gitPROpener)
		require.True(t, ok)

		res, err := runner.run(
			t.Context(),
			&promotion.StepContext{
				Project: "fake-project",
				Stage:   "fake-stage",
				WorkDir: workDir,
			},
			builtin.GitOpenPRConfig{
				RepoURL:      testRepoURL,
				SourceBranch: testSourceBranch,
				TargetBranch: testTargetBranch,
				Provider:     ptr.To(builtin.Provider(fakeGitProviderName)),
			},
		)
		require.NoError(t, err)
		require.Equal(t, kargoapi.PromotionStepStatusSkipped, res.Status)
	})

	t.Run("successfully opens PR when target branch exists and has changes", func(t *testing.T) {
		// ensure there is a change compared to the source branch by pushing a commit to the target branch
		require.NoError(t, repo.Checkout(testTargetBranch))
		require.NoError(t, os.WriteFile(fmt.Sprintf("%s/some-file.txt", repo.Dir()), []byte("some changes"), 0600))
		require.NoError(t, repo.AddAllAndCommit("my commit msg", nil))
		require.NoError(t, repo.Push(nil))

		r := newGitPROpener(promotion.StepRunnerCapabilities{
			CredsDB: &credentials.FakeDB{},
		})
		runner, ok := r.(*gitPROpener)
		require.True(t, ok)

		res, err := runner.run(
			t.Context(),
			&promotion.StepContext{
				Project: "fake-project",
				Stage:   "fake-stage",
				WorkDir: workDir,
			},
			builtin.GitOpenPRConfig{
				RepoURL:            testRepoURL,
				SourceBranch:       testSourceBranch,
				TargetBranch:       testTargetBranch,
				CreateTargetBranch: true,
				Provider:           ptr.To(builtin.Provider(fakeGitProviderName)),
				Title:              "kargo",
				Description:        "kargo description",
			},
		)
		require.NoError(t, err)
		require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)

		// Validate the pr.ID and pr.URL fields
		prOutput, ok := res.Output["pr"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, testPRNumber, prOutput["id"])
		require.Equal(t, testPRURL, prOutput["url"])
	})
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
