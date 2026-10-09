package builtin

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/credentials"
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
			name: "repoURL is an SSH URL",
			config: promotion.Config{
				"repoURL": "ssh://git@github.com/example/repo.git",
			},
			expectedProblems: []string{
				"repoURL: Does not match pattern '^https?://'",
			},
		},
		{
			name: "repoURL is an SCP-style URL",
			config: promotion.Config{
				"repoURL": "git@github.com:example/repo.git",
			},
			expectedProblems: []string{
				"repoURL: Does not match pattern '^https?://'",
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
			name: "depth is zero",
			config: promotion.Config{
				"repoURL":  "https://github.com/example/repo.git",
				"depth":    0,
				"checkout": []promotion.Config{{"path": "/fake/path/0"}},
			},
			expectedProblems: []string{
				"depth: Must be greater than or equal to 1",
			},
		},
		{
			name: "depth is negative",
			config: promotion.Config{
				"repoURL":  "https://github.com/example/repo.git",
				"depth":    -1,
				"checkout": []promotion.Config{{"path": "/fake/path/0"}},
			},
			expectedProblems: []string{
				"depth: Must be greater than or equal to 1",
			},
		},
		{
			name: "depth is valid",
			config: promotion.Config{
				"repoURL": "https://github.com/example/repo.git",
				"depth":   100,
				"checkout": []promotion.Config{{
					"branch": "main",
					"path":   "/fake/path/0",
				}},
			},
		},
		{
			name: "depth with a commit checkout",
			config: promotion.Config{
				"repoURL": "https://github.com/example/repo.git",
				"depth":   100,
				"checkout": []promotion.Config{
					{"branch": "main", "path": "/fake/path/0"},
					{"commit": "fake-commit", "path": "/fake/path/1"},
				},
			},
			expectedProblems: []string{
				"checkout[1] must specify a branch when depth or branches is specified",
			},
		},
		{
			name: "depth with a tag checkout",
			config: promotion.Config{
				"repoURL": "https://github.com/example/repo.git",
				"depth":   100,
				"checkout": []promotion.Config{
					{"tag": "fake-tag", "path": "/fake/path/0"},
				},
			},
			expectedProblems: []string{
				"checkout[0] must specify a branch when depth or branches is specified",
			},
		},
		{
			name: "branches with a default branch checkout",
			config: promotion.Config{
				"repoURL":  "https://github.com/example/repo.git",
				"branches": []string{"main"},
				"checkout": []promotion.Config{{"path": "/fake/path/0"}},
			},
			expectedProblems: []string{
				"checkout[0] must specify a branch when depth or branches is specified",
			},
		},
		{
			name: "commit checkouts are fine without depth or branches",
			config: promotion.Config{
				"repoURL": "https://github.com/example/repo.git",
				"checkout": []promotion.Config{
					{"commit": "fake-commit", "path": "/fake/path/0"},
					{"tag": "fake-tag", "path": "/fake/path/1"},
				},
			},
		},
		{
			name: "branches is an empty array",
			config: promotion.Config{
				"repoURL":  "https://github.com/example/repo.git",
				"branches": []string{},
				"checkout": []promotion.Config{{"path": "/fake/path/0"}},
			},
			expectedProblems: []string{
				"branches: Array must have at least 1 items",
			},
		},
		{
			name: "branches contains an empty string",
			config: promotion.Config{
				"repoURL":  "https://github.com/example/repo.git",
				"branches": []string{""},
				"checkout": []promotion.Config{{"path": "/fake/path/0"}},
			},
			expectedProblems: []string{
				"branches.0: String length must be greater than or equal to 1",
			},
		},
		{
			name: "branches pattern contains a colon",
			config: promotion.Config{
				"repoURL":  "https://github.com/example/repo.git",
				"branches": []string{"main:other"},
				"checkout": []promotion.Config{{"path": "/fake/path/0"}},
			},
			expectedProblems: []string{"branches.0: Does not match pattern"},
		},
		{
			name: "branches pattern contains whitespace",
			config: promotion.Config{
				"repoURL":  "https://github.com/example/repo.git",
				"branches": []string{"my branch"},
				"checkout": []promotion.Config{{"path": "/fake/path/0"}},
			},
			expectedProblems: []string{"branches.0: Does not match pattern"},
		},
		{
			name: "branches pattern has a leading plus",
			config: promotion.Config{
				"repoURL":  "https://github.com/example/repo.git",
				"branches": []string{"+main"},
				"checkout": []promotion.Config{{"path": "/fake/path/0"}},
			},
			expectedProblems: []string{"branches.0: Does not match pattern"},
		},
		{
			name: "branches pattern has a leading dash",
			config: promotion.Config{
				"repoURL":  "https://github.com/example/repo.git",
				"branches": []string{"--upload-pack=evil"},
				"checkout": []promotion.Config{{"path": "/fake/path/0"}},
			},
			expectedProblems: []string{"branches.0: Does not match pattern"},
		},
		{
			name: "branches pattern contains multiple wildcards",
			config: promotion.Config{
				"repoURL":  "https://github.com/example/repo.git",
				"branches": []string{"stage/*/*"},
				"checkout": []promotion.Config{{"path": "/fake/path/0"}},
			},
			expectedProblems: []string{"branches.0: Does not match pattern"},
		},
		{
			name: "checkout branch does not match branches",
			config: promotion.Config{
				"repoURL":  "https://github.com/example/repo.git",
				"branches": []string{"main", "stage/*"},
				"checkout": []promotion.Config{
					{"branch": "main", "path": "/fake/path/0"},
					{"branch": "feature/foo", "path": "/fake/path/1"},
				},
			},
			expectedProblems: []string{
				`branch "feature/foo" at checkout[1] does not match any of the patterns`,
			},
		},
		{
			name: "branches are valid",
			config: promotion.Config{
				"repoURL":  "https://github.com/example/repo.git",
				"depth":    10,
				"branches": []string{"main", "stage/*", "*-prod", "*"},
				"checkout": []promotion.Config{
					{"branch": "main", "path": "/fake/path/0"},
					{"branch": "stage/dev", "path": "/fake/path/1"},
					{"branch": "release-prod", "path": "/fake/path/2"},
				},
			},
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

	r := newGitCloner(promotion.StepRunnerCapabilities{})
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
	repo, err := git.Clone(t.Context(), testRepoURL, nil, nil)
	require.NoError(t, err)
	defer repo.Close(t.Context())
	err = os.WriteFile(filepath.Join(repo.Dir(), "test.txt"), []byte("foo"), 0600)
	require.NoError(t, err)
	err = repo.AddAllAndCommit(t.Context(), "Initial commit", nil)
	require.NoError(t, err)
	err = repo.Push(t.Context(), nil)
	require.NoError(t, err)

	srcBranchCommitID, err := repo.LastCommitID(t.Context())
	require.NoError(t, err)

	// Now we can proceed to test gitCloner...

	r := newGitCloner(promotion.StepRunnerCapabilities{
		CredsDB:         &credentials.FakeDB{},
		GitUserResolver: &fakeGitUserResolver{},
	})
	runner, ok := r.(*gitCloner)
	require.True(t, ok)

	stepCtx := &promotion.StepContext{
		WorkDir: t.TempDir(),
	}

	res, err := runner.run(
		t.Context(),
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
	// The checked out main branch should have the content we know is in the
	// test remote's main branch.
	require.FileExists(t, filepath.Join(stepCtx.WorkDir, "src", "test.txt"))
	require.DirExists(t, filepath.Join(stepCtx.WorkDir, "out"))
	// The stage/dev branch is a new orphan branch with a single empty commit.
	// It should lack any content.
	dirEntries, err := os.ReadDir(filepath.Join(stepCtx.WorkDir, "out"))
	require.NoError(t, err)
	require.Len(t, dirEntries, 1) // Just the .git file
	require.FileExists(t, filepath.Join(stepCtx.WorkDir, "out", ".git"))

	// Assert output map contains the expected commit hashes for each checkout
	outTree, err := git.LoadWorkTree(
		t.Context(),
		filepath.Join(stepCtx.WorkDir, "out"),
		nil,
	)
	require.NoError(t, err)
	outBranchCommitID, err := outTree.LastCommitID(t.Context())
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

func Test_branchMatchesAny(t *testing.T) {
	testCases := []struct {
		name     string
		branch   string
		patterns []string
		expected bool
	}{
		{
			name:     "no patterns",
			branch:   "main",
			expected: false,
		},
		{
			name:     "exact match",
			branch:   "main",
			patterns: []string{"main"},
			expected: true,
		},
		{
			name:     "no exact match",
			branch:   "main",
			patterns: []string{"mai", "mainx"},
			expected: false,
		},
		{
			name:     "prefix wildcard match",
			branch:   "stage/dev",
			patterns: []string{"stage/*"},
			expected: true,
		},
		{
			name:     "wildcard matches nested path",
			branch:   "stage/dev/blue",
			patterns: []string{"stage/*"},
			expected: true,
		},
		{
			name:     "wildcard matches empty string",
			branch:   "stage/",
			patterns: []string{"stage/*"},
			expected: true,
		},
		{
			name:     "suffix wildcard match",
			branch:   "dev-prod",
			patterns: []string{"*-prod"},
			expected: true,
		},
		{
			name:     "infix wildcard match",
			branch:   "release/1.0/final",
			patterns: []string{"release/*/final"},
			expected: true,
		},
		{
			name:     "prefix and suffix must not overlap",
			branch:   "aba",
			patterns: []string{"ab*ba"},
			expected: false,
		},
		{
			name:     "lone wildcard matches everything",
			branch:   "anything/goes",
			patterns: []string{"*"},
			expected: true,
		},
		{
			name:     "wildcard does not match",
			branch:   "feature/foo",
			patterns: []string{"stage/*", "*-prod"},
			expected: false,
		},
		{
			name:     "any pattern may match",
			branch:   "feature/foo",
			patterns: []string{"stage/*", "feature/*"},
			expected: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expected,
				branchMatchesAny(testCase.branch, testCase.patterns),
			)
		})
	}
}

func Test_gitCloner_run_with_depth_and_branches(t *testing.T) {
	// Set up a test Git server in-process
	service := gitkit.New(
		gitkit.Config{
			Dir:        t.TempDir(),
			AutoCreate: true,
		},
	)
	require.NoError(t, service.Setup())
	server := httptest.NewServer(service)
	t.Cleanup(server.Close)

	testRepoURL := fmt.Sprintf("%s/test.git", server.URL)

	// Create a main branch with three commits and two other branches that each
	// add one commit on top of main.
	repo, err := git.Clone(t.Context(), testRepoURL, nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = repo.Close(t.Context())
	})
	for i := range 3 {
		err = os.WriteFile(
			filepath.Join(repo.Dir(), "test.txt"),
			[]byte(fmt.Sprintf("commit %d", i)),
			0600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), fmt.Sprintf("commit %d", i), nil)
		require.NoError(t, err)
	}
	err = repo.Push(t.Context(), nil)
	require.NoError(t, err)
	for _, branch := range []string{"stage/dev", "feature/foo"} {
		err = repo.Checkout(t.Context(), "main")
		require.NoError(t, err)
		err = repo.CreateChildBranch(t.Context(), branch)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(
			t.Context(),
			fmt.Sprintf("%s commit", branch),
			&git.CommitOptions{AllowEmpty: true},
		)
		require.NoError(t, err)
		err = repo.Push(t.Context(), nil)
		require.NoError(t, err)
	}

	r := newGitCloner(promotion.StepRunnerCapabilities{
		CredsDB:         &credentials.FakeDB{},
		GitUserResolver: &fakeGitUserResolver{},
	})
	runner, ok := r.(*gitCloner)
	require.True(t, ok)

	stepCtx := &promotion.StepContext{WorkDir: t.TempDir()}
	depth := int64(2)
	res, err := runner.run(
		t.Context(),
		stepCtx,
		builtin.GitCloneConfig{
			RepoURL:  testRepoURL,
			Depth:    &depth,
			Branches: []string{"stage/*"},
			Checkout: []builtin.Checkout{{Branch: "stage/dev", Path: "out"}},
		},
	)
	require.NoError(t, err)
	require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)

	outDir := filepath.Join(stepCtx.WorkDir, "out")
	require.FileExists(t, filepath.Join(outDir, "test.txt"))

	// The history should be limited to the stage/dev tip and its parent.
	cmd := exec.CommandContext(t.Context(), "git", "rev-list", "--count", "HEAD")
	cmd.Dir = outDir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git rev-list failed: %s", out)
	require.Equal(t, "2", strings.TrimSpace(string(out)))

	// Branches that aren't covered by the specified patterns were not fetched.
	for _, branch := range []string{"main", "feature/foo"} {
		cmd = exec.CommandContext(
			t.Context(),
			"git", "rev-parse", "--verify", "--quiet", "refs/heads/"+branch,
		)
		cmd.Dir = outDir
		require.Error(t, cmd.Run(), "branch %q should not have been fetched", branch)
	}
}

func Test_gitCloner_run_with_branches_and_create(t *testing.T) {
	// Set up a test Git server in-process
	service := gitkit.New(
		gitkit.Config{
			Dir:        t.TempDir(),
			AutoCreate: true,
		},
	)
	require.NoError(t, service.Setup())
	server := httptest.NewServer(service)
	t.Cleanup(server.Close)

	testRepoURL := fmt.Sprintf("%s/test.git", server.URL)

	// Push some content to the remote's main branch
	repo, err := git.Clone(t.Context(), testRepoURL, nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = repo.Close(t.Context())
	})
	err = os.WriteFile(filepath.Join(repo.Dir(), "test.txt"), []byte("foo"), 0600)
	require.NoError(t, err)
	err = repo.AddAllAndCommit(t.Context(), "Initial commit", nil)
	require.NoError(t, err)
	err = repo.Push(t.Context(), nil)
	require.NoError(t, err)

	r := newGitCloner(promotion.StepRunnerCapabilities{
		CredsDB:         &credentials.FakeDB{},
		GitUserResolver: &fakeGitUserResolver{},
	})
	runner, ok := r.(*gitCloner)
	require.True(t, ok)

	remoteHasBranch := func(t *testing.T, branch string) bool {
		cmd := exec.CommandContext(
			t.Context(),
			"git", "ls-remote", "--heads", testRepoURL, "refs/heads/"+branch,
		)
		out, cmdErr := cmd.CombinedOutput()
		require.NoErrorf(t, cmdErr, "git ls-remote failed: %s", out)
		return strings.TrimSpace(string(out)) != ""
	}

	testCases := []struct {
		name   string
		create bool
		assert func(*testing.T, promotion.StepResult, error)
	}{
		{
			name:   "literal branch does not exist and create is false",
			create: false,
			assert: func(t *testing.T, res promotion.StepResult, err error) {
				require.ErrorContains(t, err, "does not exist")
				require.ErrorContains(t, err, "create=true")
				require.Equal(t, kargoapi.PromotionStepStatusErrored, res.Status)
				require.False(t, remoteHasBranch(t, "stage/new"))
			},
		},
		{
			name:   "literal branch does not exist and create is true",
			create: true,
			assert: func(t *testing.T, res promotion.StepResult, err error) {
				require.NoError(t, err)
				require.Equal(t, kargoapi.PromotionStepStatusSucceeded, res.Status)
				require.True(t, remoteHasBranch(t, "stage/new"))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			res, err := runner.run(
				t.Context(),
				&promotion.StepContext{WorkDir: t.TempDir()},
				builtin.GitCloneConfig{
					RepoURL:  testRepoURL,
					Branches: []string{"main", "stage/new"},
					Checkout: []builtin.Checkout{{
						Branch: "stage/new",
						Path:   "out",
						Create: testCase.create,
					}},
				},
			)
			testCase.assert(t, res, err)
		})
	}
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
	t.Cleanup(server.Close)

	// Create submodule remote repo and push a file
	subRepoURL := fmt.Sprintf("%s/sub.git", server.URL)
	subRepo, err := git.Clone(t.Context(), subRepoURL, nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = subRepo.Close(t.Context())
	})
	err = os.WriteFile(filepath.Join(subRepo.Dir(), "sub.txt"), []byte("sub"), 0o600)
	require.NoError(t, err)
	err = subRepo.AddAllAndCommit(t.Context(), "Initial commit sub", nil)
	require.NoError(t, err)
	err = subRepo.Push(t.Context(), nil)
	require.NoError(t, err)

	// Create main repo and add the submodule
	mainRepoURL := fmt.Sprintf("%s/main.git", server.URL)
	mainRepo, err := git.Clone(t.Context(), mainRepoURL, nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = mainRepo.Close(t.Context())
	})

	// Use git submodule add to create proper submodule metadata
	cmd := exec.Command("git", "submodule", "add", "-b", "main", subRepoURL, "sub")
	cmd.Dir = mainRepo.Dir()
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git submodule add failed: %s", string(out))

	// Commit and push the submodule addition
	err = mainRepo.AddAllAndCommit(t.Context(), "Add submodule", nil)
	require.NoError(t, err)
	err = mainRepo.Push(t.Context(), nil)
	require.NoError(t, err)

	mainCommitID, err := mainRepo.LastCommitID(t.Context())
	require.NoError(t, err)

	// Run git-cloner with recurseSubmodules = true
	r := newGitCloner(promotion.StepRunnerCapabilities{
		CredsDB:         &credentials.FakeDB{},
		GitUserResolver: &fakeGitUserResolver{},
	})
	runner, ok := r.(*gitCloner)
	require.True(t, ok)

	stepCtx := &promotion.StepContext{
		WorkDir: t.TempDir(),
	}

	res, err := runner.run(
		t.Context(),
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

// fakeGitUserResolver is a mock implementation of the promotion.GitUserResolver
// interface that is used to facilitate unit testing.
type fakeGitUserResolver struct {
	ResolveFn func(context.Context) (git.User, error)
}

func (f *fakeGitUserResolver) Resolve(
	ctx context.Context,
) (git.User, error) {
	if f.ResolveFn != nil {
		return f.ResolveFn(ctx)
	}
	return git.User{}, nil
}
