package builtin

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_gitCloner_convert(t *testing.T) {
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
			name:   "no checkout specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): checkout is required",
			},
		},
		{
			name: "checkout is an empty array",
			config: promotion.Config{
				"checkout": []promotion.Config{},
			},
			expectedProblems: []string{
				"checkout: Array must have at least 1 items",
			},
		},
		{
			name: "checkout path is not specified",
			config: promotion.Config{
				"checkout": []promotion.Config{{}},
			},
			expectedProblems: []string{
				"checkout.0: path is required",
			},
		},
		{
			name: "checkout path is empty string",
			config: promotion.Config{
				"checkout": []promotion.Config{{
					"path": "",
				}},
			},
			expectedProblems: []string{
				"checkout.0.path: String length must be greater than or equal to 1",
			},
		},
		{
			name: "branch and commit are both specified",
			// These are meant to be mutually exclusive.
			config: promotion.Config{
				"checkout": []promotion.Config{{
					"branch": "fake-branch",
					"commit": "fake-commit",
				}},
			},
			expectedProblems: []string{
				"checkout.0: Must validate one and only one schema",
			},
		},
		{
			name: "branch and tag are both specified",
			// These are meant to be mutually exclusive.
			config: promotion.Config{
				"checkout": []promotion.Config{{
					"branch": "fake-branch",
					"tag":    "fake-tag",
				}},
			},
			expectedProblems: []string{
				"checkout.0: Must validate one and only one schema",
			},
		},
		{
			name: "commit and tag are both specified",
			// These are meant to be mutually exclusive.
			config: promotion.Config{
				"checkout": []promotion.Config{{
					"commit": "fake-commit",
					"tag":    "fake-tag",
				}},
			},
			expectedProblems: []string{
				"checkout.0: Must validate one and only one schema",
			},
		},
		{
			name: "duplicate aliases",
			config: promotion.Config{
				"repoURL": "https://github.com/example/repo.git",
				"checkout": []promotion.Config{
					{
						"as":   "alias1",
						"path": "/fake/path/0",
					},
					{
						"as":   "alias1",
						"path": "/fake/path/1",
					},
				},
			},
			expectedProblems: []string{
				`duplicate checkout alias "alias1" at checkout[1]`,
			},
		},
		{
			name: "author name is missing",
			config: promotion.Config{
				"repoURL": "https://github.com/example/repo.git",
				"checkout": []promotion.Config{
					{
						"path": "/fake/path/0",
					},
				},
				"author": promotion.Config{
					"email": "tony@starkindustries.com",
					// Missing "name"
				},
			},
			expectedProblems: []string{
				"author: name is required",
			},
		},
		{
			name: "author name is empty",
			config: promotion.Config{
				"repoURL": "https://github.com/example/repo.git",
				"checkout": []promotion.Config{
					{
						"path": "/fake/path/0",
					},
				},
				"author": promotion.Config{
					"name":  "",
					"email": "tony@starkindustries.com",
				},
			},
			expectedProblems: []string{
				"author.name: String length must be greater than or equal to 1",
			},
		},
		{
			name: "author email is missing",
			config: promotion.Config{
				"repoURL": "https://github.com/example/repo.git",
				"checkout": []promotion.Config{
					{
						"path": "/fake/path/1",
					},
				},
				"author": promotion.Config{
					"name": "Tony Stark",
					// Missing "email"
				},
			},
			expectedProblems: []string{
				"author: email is required",
			},
		},
		{
			name: "author email is empty",
			config: promotion.Config{
				"repoURL": "https://github.com/example/repo.git",
				"checkout": []promotion.Config{
					{
						"path": "/fake/path/1",
					},
				},
				"author": promotion.Config{
					"name":  "Tony Stark",
					"email": "",
				},
			},
			expectedProblems: []string{
				"author.email: Does not match format 'email'",
			},
		},
		{
			name: "signingKey is missing",
			config: promotion.Config{
				"repoURL": "https://github.com/example/repo.git",
				"checkout": []promotion.Config{
					{
						"path": "/fake/path/2",
					},
				},
				"author": promotion.Config{
					"name":  "Tony Stark",
					"email": "tony@starkindustries.com",
					// signingKey is absent
				},
			},
			// No expected problems because signingKey is optional
		},
		{
			name: "signingKey is empty string",
			config: promotion.Config{
				"repoURL": "https://github.com/example/repo.git",
				"checkout": []promotion.Config{
					{
						"path": "/fake/path/1",
					},
				},
				"author": promotion.Config{
					"name":       "Tony Stark",
					"email":      "tony@starkindustries.com",
					"signingKey": "", // Empty string for signing key
				},
			},
			// No expected problems because signingKey is optional and empty is valid
		},
		{
			name: "valid kitchen sink",
			config: promotion.Config{
				"repoURL": "https://github.com/example/repo.git",
				"checkout": []promotion.Config{
					{
						"path": "/fake/path/0",
					},
					{
						"branch": "",
						"commit": "",
						"tag":    "",
						"path":   "/fake/path/1",
					},
					{
						"branch": "fake-branch",
						"path":   "/fake/path/2",
					},
					{
						"branch": "fake-branch",
						"commit": "",
						"tag":    "",
						"path":   "/fake/path/3",
					},
					{
						"commit": "fake-commit",
						"path":   "/fake/path/4",
					},
					{
						"branch": "",
						"commit": "fake-commit",
						"tag":    "",
						"path":   "/fake/path/5",
					},
					{
						"tag":  "fake-tag",
						"path": "/fake/path/6",
					},
					{
						"branch": "",
						"commit": "",
						"tag":    "fake-tag",
						"path":   "/fake/path/7",
					},
					{
						"path": "/fake/path/8",
						"as":   "alias1",
					},
					{
						"branch": "",
						"commit": "",
						"tag":    "",
						"path":   "/fake/path/9",
						"as":     "alias2",
					},
					{
						"path": "/fake/path/10",
					},
				},
			},
		},
	}

	r := newGitCloner(nil)
	runner, ok := r.(*gitCloner)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_gitCloner_run(t *testing.T) {
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

	// Create some content and push it to the remote repository's default branch
	repo, err := git.Clone(testRepoURL, nil, nil)
	require.NoError(t, err)
	defer repo.Close()
	err = os.WriteFile(filepath.Join(repo.Dir(), "test.txt"), []byte("foo"), 0600)
	require.NoError(t, err)
	err = repo.AddAllAndCommit("Initial commit", nil)
	require.NoError(t, err)
	err = repo.Push(nil)
	require.NoError(t, err)

	srcBranchCommitID, err := repo.LastCommitID()
	require.NoError(t, err)

	// Now we can proceed to test gitCloner...

	r := newGitCloner(&credentials.FakeDB{})
	runner, ok := r.(*gitCloner)
	require.True(t, ok)

	stepCtx := &promotion.StepContext{
		WorkDir: t.TempDir(),
	}

	res, err := runner.run(
		context.Background(),
		stepCtx,
		builtin.GitCloneConfig{
			RepoURL: fmt.Sprintf("%s/test.git", server.URL),
			Checkout: []builtin.Checkout{
				{
					Commit: srcBranchCommitID,
					Path:   "src",
				},
				{
					Branch: "stage/dev",
					Path:   "out",
					Create: true,
				},
			},
		},
	)
	require.NoError(t, err)
	require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
	require.DirExists(t, filepath.Join(stepCtx.WorkDir, "src"))
	// The checked out master branch should have the content we know is in the
	// test remote's master branch.
	require.FileExists(t, filepath.Join(stepCtx.WorkDir, "src", "test.txt"))
	require.DirExists(t, filepath.Join(stepCtx.WorkDir, "out"))
	// The stage/dev branch is a new orphan branch with a single empty commit.
	// It should lack any content.
	dirEntries, err := os.ReadDir(filepath.Join(stepCtx.WorkDir, "out"))
	require.NoError(t, err)
	require.Len(t, dirEntries, 1) // Just the .git file
	require.FileExists(t, filepath.Join(stepCtx.WorkDir, "out", ".git"))

	// Assert output map contains the expected commit hashes for each checkout
	outTree, err := git.LoadWorkTree(filepath.Join(stepCtx.WorkDir, "out"), nil)
	require.NoError(t, err)
	outBranchCommitID, err := outTree.LastCommitID()
	require.NoError(t, err)
	require.Equal(
		t,
		map[string]any{
			"src": srcBranchCommitID,
			"out": outBranchCommitID,
		},
		res.Output["commits"],
	)
}

func Test_gitCloner_run_with_submodules(t *testing.T) {
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

	// Create submodule remote repo and push a file
	subRepoURL := fmt.Sprintf("%s/sub.git", server.URL)
	subRepo, err := git.Clone(subRepoURL, nil, nil)
	require.NoError(t, err)
	defer subRepo.Close()
	err = os.WriteFile(filepath.Join(subRepo.Dir(), "sub.txt"), []byte("sub"), 0600)
	require.NoError(t, err)
	err = subRepo.AddAllAndCommit("Initial commit sub", nil)
	require.NoError(t, err)
	err = subRepo.Push(nil)
	require.NoError(t, err)

	// Create main repo and add the submodule
	mainRepoURL := fmt.Sprintf("%s/main.git", server.URL)
	mainRepo, err := git.Clone(mainRepoURL, nil, nil)
	require.NoError(t, err)
	defer mainRepo.Close()

	// Use git submodule add to create proper submodule metadata
	cmd := exec.Command("git", "submodule", "add", subRepoURL, "sub")
	cmd.Dir = mainRepo.Dir()
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git submodule add failed: %s", string(out))

	// Commit and push the submodule addition
	err = mainRepo.AddAllAndCommit("Add submodule", nil)
	require.NoError(t, err)
	err = mainRepo.Push(nil)
	require.NoError(t, err)

	mainCommitID, err := mainRepo.LastCommitID()
	require.NoError(t, err)

	// Run git-cloner with recurseSubmodules = true
	r := newGitCloner(&credentials.FakeDB{})
	runner, ok := r.(*gitCloner)
	require.True(t, ok)

	stepCtx := &promotion.StepContext{
		WorkDir: t.TempDir(),
	}

	res, err := runner.run(
		context.Background(),
		stepCtx,
		builtin.GitCloneConfig{
			RepoURL:           mainRepoURL,
			Checkout:          []builtin.Checkout{{Commit: mainCommitID, Path: "src"}},
			RecurseSubmodules: true,
		},
	)
	require.NoError(t, err)
	require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)

	// Assert submodule file was populated inside worktree
	require.FileExists(t, filepath.Join(stepCtx.WorkDir, "src", "sub", "sub.txt"))
}
