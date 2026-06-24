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
// Setup requirements (env vars and the branch protection needed by the
// "blocked" subtest) are documented in pkg/gitprovider/testing/README.md. The
// "blocked" subtest skips unless TEST_GITHUB_REQUIRE_STATUS_CHECK=true and the
// repo's main branch requires a status check the PR will not satisfy.
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

	// A required status check is repo-global, so it blocks every PR -- including
	// the clean one. The "clean"/"dirty" subtests therefore assume an unprotected
	// main, while "blocked" assumes the required check is configured. The two
	// modes are mutually exclusive; select with TEST_GITHUB_REQUIRE_STATUS_CHECK.
	protected := os.Getenv("TEST_GITHUB_REQUIRE_STATUS_CHECK") == "true"

	t.Run("clean merges", func(t *testing.T) {
		if protected {
			t.Skip("repo has a required status check; clean assumes unprotected main")
		}
		prNumber, cleanup := gptest.SetupCleanPR(t, repoCfg, prov)
		defer cleanup()

		ghPR := waitForMergeableComputed(t, prov, prNumber)
		require.Equal(t, "clean", ghPR.GetMergeableState())

		mergedPR, merged, mergeErr := prov.MergePullRequest(t.Context(), prNumber, nil)
		require.NoError(t, mergeErr)
		require.True(t, merged)
		require.NotNil(t, mergedPR)
		logGateComparison(t, prNumber, ghPR, gateMerge)
	})

	t.Run("dirty fails with conflict", func(t *testing.T) {
		if protected {
			t.Skip("repo has a required status check; dirty assumes unprotected main")
		}
		prNumber, cleanup := gptest.SetupConflictingPR(t, repoCfg, prov)
		defer cleanup()

		ghPR := waitForMergeableComputed(t, prov, prNumber)
		require.Equal(t, "dirty", ghPR.GetMergeableState(), "expected GitHub to report a conflict")

		mergedPR, merged, mergeErr := prov.MergePullRequest(t.Context(), prNumber, nil)
		require.Error(t, mergeErr)
		require.Contains(t, mergeErr.Error(), "has conflicts and cannot be merged")
		require.False(t, merged)
		require.Nil(t, mergedPR)
		logGateComparison(t, prNumber, ghPR, gateFail)
	})

	t.Run("blocked is not ready", func(t *testing.T) {
		if !protected {
			t.Skip(
				"set TEST_GITHUB_REQUIRE_STATUS_CHECK=true with the repo's main " +
					"branch requiring a status check the PR will not satisfy",
			)
		}
		prNumber, cleanup := gptest.SetupCleanPR(t, repoCfg, prov)
		defer cleanup()

		// With an unsatisfied required status check, GitHub blocks the merge.
		ghPR := waitForMergeableComputed(t, prov, prNumber)
		require.Equal(t, "blocked", ghPR.GetMergeableState(), "expected GitHub to block the merge")

		mergedPR, merged, mergeErr := prov.MergePullRequest(t.Context(), prNumber, nil)
		require.NoError(t, mergeErr)
		require.False(t, merged)
		require.Nil(t, mergedPR)
		logGateComparison(t, prNumber, ghPR, gateRetry)
	})
}

// Gate outcomes, used to compare the new mergeable_state gate against the legacy
// Mergeable-boolean gate.
const (
	gateMerge = "merge"
	gateFail  = "fail (terminal error)"
	gateRetry = "not-ready (retry)"
)

// logGateComparison records, for one PR, the inputs GitHub returned, the outcome
// the legacy gate (Draft || !Mergeable) would have produced, and the new gate's
// observed outcome -- flagging where the added mergeable_state detail changes
// behavior. The legacy gate can only ever merge or retry; it cannot distinguish
// a permanent conflict (so it would retry "dirty" forever) and trusts the
// Mergeable boolean (so it would attempt to merge a "blocked" PR that GitHub
// then rejects).
func logGateComparison(
	t *testing.T,
	prNumber int64,
	ghPR github.PullRequest,
	newGate string,
) {
	t.Helper()
	legacy := gateMerge
	if ghPR.GetDraft() || !ghPR.GetMergeable() {
		legacy = gateRetry
	}
	verdict := "same as legacy"
	if legacy != newGate {
		verdict = "DIVERGES from legacy"
	}
	t.Logf(
		"PR #%d: mergeable_state=%q mergeable=%v draft=%v",
		prNumber, ghPR.GetMergeableState(), ghPR.GetMergeable(), ghPR.GetDraft(),
	)
	t.Logf("  legacy boolean gate -> %s", legacy)
	t.Logf("  new gate            -> %s  (%s)", newGate, verdict)
}

// waitForMergeableComputed polls the PR until GitHub has finished computing its
// mergeability (mergeable_state is neither empty nor "unknown") and returns the
// resolved PR. GitHub computes mergeability asynchronously, so reading too early
// would observe "unknown" and mask the state the test is exercising.
func waitForMergeableComputed(
	t *testing.T,
	prov gitprovider.Interface,
	prNumber int64,
) github.PullRequest {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		pr, err := prov.GetPullRequest(t.Context(), prNumber)
		require.NoError(t, err)
		ghPR, ok := pr.Object.(github.PullRequest)
		require.True(t, ok, "expected PR object to be a github.PullRequest")
		if state := ghPR.GetMergeableState(); state != "" && state != "unknown" {
			return ghPR
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for mergeable_state of PR %d", prNumber)
		}
		time.Sleep(2 * time.Second)
	}
}
