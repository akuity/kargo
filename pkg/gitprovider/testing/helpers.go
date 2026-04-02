//go:build integration

// Package testing provides shared helpers for gitprovider integration tests.
package testing

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/controller/git"
	"github.com/akuity/kargo/pkg/gitprovider"
)

// RequireEnv returns the value of the named environment variable, or skips the
// test if it is not set.
func RequireEnv(t *testing.T, name string) string {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		t.Skipf("%s must be set", name)
	}
	return v
}

// RepoConfig holds the provider-specific constants needed by the shared test
// helpers.
type RepoConfig struct {
	RepoURL     string
	Token       string
	GitUsername string
	// AuthedRepoURL is the clone URL with credentials embedded. Providers that
	// embed credentials differently (e.g. Azure DevOps) must set this
	// explicitly. If empty, it is derived as
	// https://<GitUsername>:<Token>@<host>/<path>.
	AuthedRepoURL string
	// MergeWaitDuration is how long to wait after creating a PR before
	// attempting to merge. Some providers (e.g. GitLab) need longer to compute
	// mergeability. Defaults to 5 seconds if zero.
	MergeWaitDuration time.Duration
}

func (c RepoConfig) clientOpts() *git.ClientOptions {
	return &git.ClientOptions{
		Credentials: &git.RepoCredentials{
			Username: c.GitUsername,
			Password: c.Token,
		},
		User: &git.User{
			Name:  "Kargo Integration Test",
			Email: "test@kargo.io",
		},
	}
}

func (c RepoConfig) authedRepoURL() string {
	if c.AuthedRepoURL != "" {
		return c.AuthedRepoURL
	}
	const prefix = "https://"
	if !strings.HasPrefix(c.RepoURL, prefix) {
		return c.RepoURL
	}
	return fmt.Sprintf(
		"%s%s:%s@%s",
		prefix, c.GitUsername, c.Token,
		c.RepoURL[len(prefix):],
	)
}

func (c RepoConfig) mergeWait() time.Duration {
	if c.MergeWaitDuration > 0 {
		return c.MergeWaitDuration
	}
	return 5 * time.Second
}

// PRTestCase defines a single test case for the PR integration test.
type PRTestCase struct {
	Name            string
	MergeMethod     string
	ExpectedParents int
	ExpectMergeErr  bool
}

// RunPRTests is the shared test runner for gitprovider integration tests. It
// exercises CreatePullRequest and MergePullRequest for each test case.
func RunPRTests(
	t *testing.T,
	cfg RepoConfig,
	prov gitprovider.Interface,
	testCases []PRTestCase,
) {
	t.Helper()
	ensureMainBranch(t, cfg)

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			branchName := uniqueBranchName(tc.Name)
			repo := cloneAndPush(t, cfg, branchName)
			defer cleanupBranch(repo, branchName)

			pr, err := prov.CreatePullRequest(
				t.Context(),
				&gitprovider.CreatePullRequestOpts{
					Title: fmt.Sprintf("integration test: %s", tc.Name),
					Head:  branchName,
					Base:  "main",
				},
			)
			require.NoError(t, err)
			require.NotNil(t, pr)
			require.True(t, pr.Open)

			// Give the provider time to compute mergeability -- most of them
			// do so asynchronously.
			time.Sleep(cfg.mergeWait())

			var mergeOpts *gitprovider.MergePullRequestOpts
			if tc.MergeMethod != "" {
				mergeOpts = &gitprovider.MergePullRequestOpts{
					MergeMethod: tc.MergeMethod,
				}
			}

			mergedPR, merged, mergeErr := prov.MergePullRequest(
				t.Context(), pr.Number, mergeOpts,
			)
			if tc.ExpectMergeErr {
				require.Error(t, mergeErr)
				require.False(t, merged)
				return
			}
			require.NoError(t, mergeErr)
			require.True(t, merged)

			fetchMain(t, cfg, repo)
			requireParentCount(
				t, repo, mergedPR.MergeCommitSHA, tc.ExpectedParents,
			)
		})
	}
}

// ensureMainBranch ensures the test repo has a main branch with at least one
// commit. This is idempotent — it's a no-op if main already exists.
func ensureMainBranch(t *testing.T, cfg RepoConfig) {
	t.Helper()
	if _, err := git.Clone(
		cfg.RepoURL,
		cfg.clientOpts(),
		&git.CloneOptions{Branch: "main", SingleBranch: true},
	); err == nil {
		return // main exists
	}
	// Empty repo — initialize main.
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "--initial-branch", "main"},
		{"config", "user.name", "Test"},
		{"config", "user.email", "test@test.com"},
	} {
		runGit(t, dir, args...)
	}
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "README.md"), []byte("# test-repo\n"), 0600,
	))
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "--no-gpg-sign", "-m", "initial commit")
	runGit(t, dir, "remote", "add", "origin", cfg.authedRepoURL())
	runGit(t, dir, "push", "-u", "origin", "main")
}

// uniqueBranchName returns a branch name that is unique and safe for use as a
// git ref.
func uniqueBranchName(testName string) string {
	safeName := strings.ReplaceAll(testName, " ", "-")
	return fmt.Sprintf(
		"integration-test-%s-%d", safeName, time.Now().UnixNano(),
	)
}

// cloneAndPush clones the test repo, creates a feature branch with a trivial
// commit, and pushes it.
func cloneAndPush(
	t *testing.T, cfg RepoConfig, branchName string,
) git.Repo {
	t.Helper()
	repo, err := git.Clone(
		cfg.RepoURL,
		cfg.clientOpts(),
		&git.CloneOptions{Branch: "main", SingleBranch: true},
	)
	require.NoError(t, err)
	require.NoError(t, repo.CreateChildBranch(branchName))
	require.NoError(t, os.WriteFile(
		filepath.Join(repo.Dir(), fmt.Sprintf("test-%s.txt", branchName)),
		[]byte(fmt.Sprintf("test content %d", time.Now().UnixNano())),
		0600,
	))
	require.NoError(t, repo.AddAllAndCommit(
		fmt.Sprintf("test commit for %s", branchName), nil,
	))
	require.NoError(t, repo.Push(nil))
	return repo
}

// fetchMain fetches the latest main branch from the remote.
func fetchMain(t *testing.T, cfg RepoConfig, repo git.Repo) {
	t.Helper()
	cmd := exec.Command("git", "fetch", "origin", "main")
	cmd.Dir = repo.Dir()
	cmd.Env = append(
		os.Environ(),
		fmt.Sprintf("HOME=%s", repo.HomeDir()),
		"GIT_ASKPASS=/usr/local/bin/credential-helper",
		fmt.Sprintf("GIT_PASSWORD=%s", cfg.Token),
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

// requireParentCount asserts that the given commit has the expected number of
// parents. This is used to perform some minimal validation of the merge method
// that was actually used. Two parents indicates a merge commit, while one
// parent indicates a rebase or squash. Note that this is not a perfect test —
// for example, a rebase could produce multiple commits if there are multiple
// commits on the branch, and a merge commit could have only one parent if the
// source branch had only one commit. However, in the context of these tests
// where we create a single commit on the feature branch, it should be
// sufficient to distinguish between merge vs. rebase/squash.
func requireParentCount(
	t *testing.T, repo git.Repo, sha string, expected int,
) {
	t.Helper()
	require.NotEmpty(t, sha, "merge commit SHA must not be empty")
	cmd := exec.Command("git", "cat-file", "-p", sha)
	cmd.Dir = repo.Dir()
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	parentCount := 0
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "parent ") {
			parentCount++
		}
	}
	require.Equal(t, expected, parentCount,
		"commit %s: expected %d parent(s), got %d", sha, expected, parentCount,
	)
}

// cleanupBranch deletes the remote branch and closes the repo. Best-effort.
func cleanupBranch(repo git.Repo, branchName string) {
	cmd := exec.Command("git", "push", "origin", "--delete", branchName)
	cmd.Dir = repo.Dir()
	cmd.Env = append(
		os.Environ(), fmt.Sprintf("HOME=%s", repo.HomeDir()),
	)
	_ = cmd.Run()
	repo.Close() // nolint: errcheck
}

// runGit runs a git command in the given directory and fails the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}
