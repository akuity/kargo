//go:build integration && github

package github

import (
	"os"
	"testing"
	"time"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/pkg/gitprovider"
	gptest "github.com/akuity/kargo/pkg/gitprovider/testing"
)

func TestCreateAndMergePullRequest(t *testing.T) {
	repoURL := gptest.RequireEnv(t, "TEST_GITHUB_REPO_URL")
	token := gptest.RequireEnv(t, "TEST_GITHUB_TOKEN")

	repoCfg := gptest.RepoConfig{
		RepoURL:     repoURL,
		Token:       token,
		GitUsername: gptest.RequireEnv(t, "TEST_GITHUB_USERNAME"),
	}

	prov, err := NewProvider(repoURL, &gitprovider.Options{Token: token})
	require.NoError(t, err)

	gptest.RunPRTests(t, repoCfg, prov, []gptest.PRTestCase{
		{
			Name:            "unspecified merge method",
			ExpectedParents: 2, // Default is merge commit
		},
		{
			Name:            "merge",
			MergeMethod:     "merge",
			ExpectedParents: 2,
		},
		{
			Name:            "rebase",
			MergeMethod:     "rebase",
			ExpectedParents: 1,
		},
		{
			Name:            "squash",
			MergeMethod:     "squash",
			ExpectedParents: 1,
		},
		{
			Name:           "invalid merge method",
			MergeMethod:    "bogus",
			ExpectMergeErr: true,
		},
	})
}

// TestMergeGate exercises the mergeable_state-aware gate in MergePullRequest
// against real GitHub PRs, confirming that GitHub reports the states the gate
// relies on.
//
// Setup requirements (env vars, host credential-helper, and the branch
// protection needed by the "behind" subtest) are documented in
// pkg/gitprovider/testing/README.md. The "behind" subtest skips unless
// TEST_GITHUB_REQUIRE_UP_TO_DATE=true and the repo's main branch requires
// branches to be up to date before merging.
func TestMergeGate(t *testing.T) {
	repoURL := gptest.RequireEnv(t, "TEST_GITHUB_REPO_URL")
	token := gptest.RequireEnv(t, "TEST_GITHUB_TOKEN")

	repoCfg := gptest.RepoConfig{
		RepoURL:     repoURL,
		Token:       token,
		GitUsername: gptest.RequireEnv(t, "TEST_GITHUB_USERNAME"),
	}

	prov, err := NewProvider(repoURL, &gitprovider.Options{Token: token})
	require.NoError(t, err)

	t.Run("clean merges", func(t *testing.T) {
		prNumber, cleanup := gptest.SetupCleanPR(t, repoCfg, prov)
		defer cleanup()

		waitForMergeableComputed(t, prov, prNumber)

		mergedPR, merged, mergeErr := prov.MergePullRequest(t.Context(), prNumber, nil)
		require.NoError(t, mergeErr)
		require.True(t, merged)
		require.NotNil(t, mergedPR)
	})

	t.Run("dirty fails with conflict", func(t *testing.T) {
		prNumber, cleanup := gptest.SetupConflictingPR(t, repoCfg, prov)
		defer cleanup()

		state := waitForMergeableComputed(t, prov, prNumber)
		require.Equal(t, "dirty", state, "expected GitHub to report a conflict")

		mergedPR, merged, mergeErr := prov.MergePullRequest(t.Context(), prNumber, nil)
		require.Error(t, mergeErr)
		require.Contains(t, mergeErr.Error(), "has conflicts and cannot be merged")
		require.False(t, merged)
		require.Nil(t, mergedPR)
	})

	t.Run("behind is not ready", func(t *testing.T) {
		if os.Getenv("TEST_GITHUB_REQUIRE_UP_TO_DATE") != "true" {
			t.Skip(
				"TEST_GITHUB_REQUIRE_UP_TO_DATE must be true and the repo's main " +
					"branch must require branches to be up to date before merging",
			)
		}
		prNumber, cleanup := gptest.SetupBehindPR(t, repoCfg, prov)
		defer cleanup()

		state := waitForMergeableComputed(t, prov, prNumber)
		require.Equal(t, "behind", state, "expected GitHub to report an out-of-date branch")

		mergedPR, merged, mergeErr := prov.MergePullRequest(t.Context(), prNumber, nil)
		require.NoError(t, mergeErr)
		require.False(t, merged)
		require.Nil(t, mergedPR)
	})
}

// waitForMergeableComputed polls the PR until GitHub has finished computing its
// mergeability (mergeable_state is neither empty nor "unknown") and returns the
// resolved state. GitHub computes mergeability asynchronously, so reading too
// early would observe "unknown" and mask the state the test is exercising.
func waitForMergeableComputed(
	t *testing.T,
	prov gitprovider.Interface,
	prNumber int64,
) string {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		pr, err := prov.GetPullRequest(t.Context(), prNumber)
		require.NoError(t, err)
		ghPR, ok := pr.Object.(github.PullRequest)
		require.True(t, ok, "expected PR object to be a github.PullRequest")
		if state := ghPR.GetMergeableState(); state != "" && state != "unknown" {
			return state
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for mergeable_state of PR %d", prNumber)
		}
		time.Sleep(2 * time.Second)
	}
}
