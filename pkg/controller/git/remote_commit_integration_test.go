package git

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	libExec "github.com/akuity/kargo/pkg/exec"
)

func Test_workTree_IntegrateRemoteChanges(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(
		t,
		initialMainCommit,
		// Push an extra commit to "ahead."
		commitAhead,
	)
	defer testServer.Close()

	t.Run("AlwaysRebase: rebases unconditionally", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		err = os.WriteFile(fmt.Sprintf("%s/local.txt", repo.Dir()), []byte("local"), 0o600)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("local commit", nil)
		require.NoError(t, err)

		// Enable signing so that RebaseOrMerge would fall back to merge, but
		// AlwaysRebase should still rebase.
		wt := internalWorkTree(t, repo)
		enableFakeCommitSigning(t, wt, true)

		err = wt.IntegrateRemoteChanges(&IntegrationOptions{
			TargetBranch:      "ahead",
			IntegrationPolicy: PushIntegrationPolicyAlwaysRebase,
		})
		require.NoError(t, err)

		// A rebase produces no merge commits:
		// initial + remote + local = 3.
		commits, err := repo.ListCommits(nil)
		require.NoError(t, err)
		require.Len(t, commits, 3)
	})

	t.Run("RebaseOrMerge: rebases when safe", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		err = os.WriteFile(fmt.Sprintf("%s/local.txt", repo.Dir()), []byte("local"), 0o600)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("local commit", nil)
		require.NoError(t, err)

		err = internalWorkTree(t, repo).IntegrateRemoteChanges(&IntegrationOptions{
			TargetBranch:      "ahead",
			IntegrationPolicy: PushIntegrationPolicyRebaseOrMerge,
		})
		require.NoError(t, err)

		// All local commits are unsigned and signing is not configured, so
		// rebase is safe. No merge commits: initial + remote + local = 3.
		commits, err := repo.ListCommits(nil)
		require.NoError(t, err)
		require.Len(t, commits, 3)
	})

	t.Run("RebaseOrMerge: merges when rebase is unsafe", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		err = os.WriteFile(fmt.Sprintf("%s/local.txt", repo.Dir()), []byte("local"), 0o600)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("local commit", nil)
		require.NoError(t, err)

		// Enable signing so that rebasing unsigned commits would be unsafe.
		wt := internalWorkTree(t, repo)
		enableFakeCommitSigning(t, wt, true)

		err = wt.IntegrateRemoteChanges(&IntegrationOptions{
			TargetBranch:      "ahead",
			IntegrationPolicy: PushIntegrationPolicyRebaseOrMerge,
		})
		require.NoError(t, err)

		// Should have fallen back to merge.
		msg, err := repo.CommitMessage("HEAD")
		require.NoError(t, err)
		require.Contains(t, msg, "Merge")
	})

	t.Run("RebaseOrFail: rebases when safe", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		err = os.WriteFile(fmt.Sprintf("%s/local.txt", repo.Dir()), []byte("local"), 0o600)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("local commit", nil)
		require.NoError(t, err)

		err = internalWorkTree(t, repo).IntegrateRemoteChanges(&IntegrationOptions{
			TargetBranch:      "ahead",
			IntegrationPolicy: PushIntegrationPolicyRebaseOrFail,
		})
		require.NoError(t, err)

		// Rebase is safe (unsigned, signing not configured).
		commits, err := repo.ListCommits(nil)
		require.NoError(t, err)
		require.Len(t, commits, 3)
	})

	t.Run("RebaseOrFail: fails when rebase is unsafe", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		err = os.WriteFile(fmt.Sprintf("%s/local.txt", repo.Dir()), []byte("local"), 0o600)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("local commit", nil)
		require.NoError(t, err)

		wt := internalWorkTree(t, repo)
		enableFakeCommitSigning(t, wt, true)

		err = wt.IntegrateRemoteChanges(&IntegrationOptions{
			TargetBranch:      "ahead",
			IntegrationPolicy: PushIntegrationPolicyRebaseOrFail,
		})
		require.ErrorIs(t, err, ErrRebaseUnsafe)
	})

	t.Run("AlwaysMerge: merges unconditionally", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		err = os.WriteFile(fmt.Sprintf("%s/local.txt", repo.Dir()), []byte("local"), 0o600)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("local commit", nil)
		require.NoError(t, err)

		err = internalWorkTree(t, repo).IntegrateRemoteChanges(&IntegrationOptions{
			TargetBranch:      "ahead",
			IntegrationPolicy: PushIntegrationPolicyAlwaysMerge,
		})
		require.NoError(t, err)

		// Even though rebase would be safe (unsigned, no signing), the policy
		// says always merge.
		msg, err := repo.CommitMessage("HEAD")
		require.NoError(t, err)
		require.Contains(t, msg, "Merge")
	})
}

func Test_workTree_pullRebase(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(
		t,
		initialMainCommit,
		// Push an extra commit to "ahead."
		commitAhead,
	)
	defer testServer.Close()

	t.Run("rebase fails with merge conflicts", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		// Add a conflicting local commit.
		err = os.WriteFile(
			fmt.Sprintf("%s/remote.txt", repo.Dir()),
			[]byte("from-local"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("local conflict", nil)
		require.NoError(t, err)

		// With a conflict, the rebase should have failed.
		err = internalWorkTree(t, repo).pullRebase("ahead")
		require.ErrorIs(t, err, ErrMergeConflict)
	})

	t.Run("rebases succeeds", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		// Add a non-conflicting local commit.
		err = os.WriteFile(
			fmt.Sprintf("%s/local.txt", repo.Dir()),
			[]byte("local"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("local commit", nil)
		require.NoError(t, err)

		// With no conflicts, the rebase should have succeeded.
		err = internalWorkTree(t, repo).pullRebase("ahead")
		require.NoError(t, err)

		// The rebase should not have created any merge commits, so the commit count
		// should be: initial + remote + local = 3.
		commits, err := repo.ListCommits(nil)
		require.NoError(t, err)
		require.Len(t, commits, 3)
	})
}

func Test_workTree_pullMerge(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(
		t,
		initialMainCommit,
		// Push an extra commit to "ahead."
		commitAhead,
	)
	defer testServer.Close()

	t.Run("merge fails with merge conflicts", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		// Add a conflicting local commit.
		err = os.WriteFile(
			fmt.Sprintf("%s/remote.txt", repo.Dir()),
			[]byte("from-local"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("local conflict", nil)
		require.NoError(t, err)

		// With a conflict, the merge should have failed.
		err = internalWorkTree(t, repo).pullMerge("ahead")
		require.ErrorIs(t, err, ErrMergeConflict)
	})

	t.Run("merge succeeds", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		// Add a non-conflicting local commit.
		err = os.WriteFile(
			fmt.Sprintf("%s/local.txt", repo.Dir()),
			[]byte("local"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("local commit", nil)
		require.NoError(t, err)

		// With no conflicts, the merge should have succeeded.
		require.NoError(t, internalWorkTree(t, repo).pullMerge("ahead"))

		// After a merge, the head of the local branch should be a merge commit.
		msg, err := repo.CommitMessage("HEAD")
		require.NoError(t, err)
		require.Contains(t, msg, "Merge")
	})
}

func Test_workTree_canSafelyRebase(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(t, initialMainCommit)
	defer testServer.Close()

	t.Run("no commits to replay", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		safe, err := internalWorkTree(t, repo).canSafelyRebase("main")
		require.NoError(t, err)
		require.True(t, safe)
	})

	t.Run("unsigned local commits to replay; signing not configured", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		// Add an unsigned local commit.
		err = os.WriteFile(
			fmt.Sprintf("%s/file.txt", repo.Dir()),
			[]byte("data"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("local commit", nil)
		require.NoError(t, err)

		safe, err := internalWorkTree(t, repo).canSafelyRebase("main")
		require.NoError(t, err)
		// This should be safe because rebasing won't lend Kargo's signature to
		// commits that were previously unsigned.
		require.True(t, safe)
	})

	t.Run("unsigned commits to replay; signing configured", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		require.NoError(t, os.WriteFile(
			fmt.Sprintf("%s/file.txt", repo.Dir()),
			[]byte("data"),
			0o600,
		))
		err = repo.AddAllAndCommit("local commit", nil)
		require.NoError(t, err)

		// Enable signing without actually importing a key.
		wt := internalWorkTree(t, repo)
		_, err = libExec.Exec(wt.buildGitCommand(
			"config", "--global", "commit.gpgSign", "true",
		))
		require.NoError(t, err)

		safe, err := wt.canSafelyRebase("main")
		require.NoError(t, err)
		// This should be unsafe because rebasing would lend Kargo's signature to
		// commits that were previously unsigned.
		require.False(t, safe)
	})

	t.Run("trusted, signed local commits to replay; signing configured", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		wt := internalWorkTree(t, repo)
		enableFakeCommitSigning(t, wt, true)

		err = os.WriteFile(
			fmt.Sprintf("%s/file.txt", repo.Dir()),
			[]byte("data"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("signed commit", nil)
		require.NoError(t, err)

		safe, err := wt.canSafelyRebase("main")
		require.NoError(t, err)
		// This should be safe because Kargo trusted the signature on commits it
		// will re-sign during the rebase.
		require.True(t, safe)
	})

	t.Run("trusted, signed local commits to replay; signing not configured", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		wt := internalWorkTree(t, repo)
		enableFakeCommitSigning(t, wt, true)

		err = os.WriteFile(
			fmt.Sprintf("%s/file.txt", repo.Dir()),
			[]byte("data"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("signed commit", nil)
		require.NoError(t, err)

		disableFakeCommitSigning(t, wt)

		safe, err := wt.canSafelyRebase("main")
		require.NoError(t, err)
		// This should be unsafe because rebasing would strip trusted signatures
		// from existing commits.
		require.False(t, safe)
	})

	t.Run("untrusted, signed commits", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		wt := internalWorkTree(t, repo)
		enableFakeCommitSigning(t, wt, false)

		err = os.WriteFile(
			fmt.Sprintf("%s/file.txt", repo.Dir()),
			[]byte("data"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("untrusted signed commit", nil)
		require.NoError(t, err)

		// Unsafe regardless — Kargo can't vouch for this commit.
		safe, err := wt.canSafelyRebase("main")
		require.NoError(t, err)
		// This should be unsafe, no matter what. If signing were not configured,
		// rebasing would strip signatures from existing commits. Something
		// downstream might trust those signatures. If signing were configured,
		// rebasing would lend Kargo's signature to commits it did not, itself,
		// trust.
		require.False(t, safe)
	})
}

func Test_workTree_commitsToReplay(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(
		t,
		initialMainCommit,
		// Push an extra commit to "ahead."
		commitAhead,
	)
	defer testServer.Close()

	t.Run("local branch and remote target branch are identical", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		commits, err := internalWorkTree(t, repo).commitsToReplay("main")
		require.NoError(t, err)
		require.Empty(t, commits)
	})

	t.Run("local branch is behind remote target branch", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		commits, err := internalWorkTree(t, repo).commitsToReplay("ahead")
		require.NoError(t, err)
		require.Empty(t, commits)
	})

	t.Run("local branch is ahead of remote target branch", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		err = os.WriteFile(
			fmt.Sprintf("%s/a.txt", repo.Dir()),
			[]byte("a"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("commit a", nil)
		require.NoError(t, err)
		err = os.WriteFile(
			fmt.Sprintf("%s/b.txt", repo.Dir()),
			[]byte("b"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("commit b", nil)
		require.NoError(t, err)

		commits, err := internalWorkTree(t, repo).commitsToReplay("main")
		require.NoError(t, err)
		// Two local commits should need to be replayed on main.
		require.Len(t, commits, 2)
	})

	t.Run("local branch and remote target branch have diverged", func(t *testing.T) {
		repo, err := Clone(testRepoURL, &ClientOptions{Credentials: &testRepoCreds}, nil)
		require.NoError(t, err)
		defer repo.Close()

		err = os.WriteFile(
			fmt.Sprintf("%s/a.txt", repo.Dir()),
			[]byte("a"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("commit a", nil)
		require.NoError(t, err)
		err = os.WriteFile(
			fmt.Sprintf("%s/b.txt", repo.Dir()),
			[]byte("b"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit("commit b", nil)
		require.NoError(t, err)

		commits, err := internalWorkTree(t, repo).commitsToReplay("ahead")
		// Two local commits should need to be replayed on ahead.
		require.NoError(t, err)
		require.Len(t, commits, 2)
	})
}

func Test_workTree_isRebaseSafeForCommit(t *testing.T) {
	testCases := []struct {
		name     string
		signing  bool
		status   signatureStatus
		expected bool
	}{
		{
			name:     "signing on, trusted",
			signing:  true,
			status:   signatureTrusted,
			expected: true,
		},
		{
			name:     "signing off, unsigned",
			signing:  false,
			status:   signatureUnsigned,
			expected: true,
		},
		{
			name:     "signing off, trusted would strip signature",
			signing:  false,
			status:   signatureTrusted,
			expected: false,
		},
		{
			name:     "signing on, untrusted",
			signing:  true,
			status:   signatureUntrusted,
			expected: false,
		},
		{
			name:     "signing off, untrusted",
			signing:  false,
			status:   signatureUntrusted,
			expected: false,
		},
		{
			name:     "signing on, unsigned would fabricate signature",
			signing:  true,
			status:   signatureUnsigned,
			expected: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := isRebaseSafeForCommit(testCase.signing, testCase.status)
			require.Equal(t, testCase.expected, result)
		})
	}
}
