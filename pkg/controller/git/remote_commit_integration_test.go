package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		err = os.WriteFile(fmt.Sprintf("%s/local.txt", repo.Dir()), []byte("local"), 0o600)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "local commit", nil)
		require.NoError(t, err)

		// Enable signing so that RebaseOrMerge would fall back to merge, but
		// AlwaysRebase should still rebase.
		wt := internalWorkTree(t, repo)
		enableFakeCommitSigning(t, wt, true)

		err = wt.IntegrateRemoteChanges(t.Context(), &IntegrationOptions{
			TargetBranch:      "ahead",
			IntegrationPolicy: PushIntegrationPolicyAlwaysRebase,
		})
		require.NoError(t, err)

		// A rebase produces no merge commits:
		// initial + remote + local = 3.
		commits, err := repo.ListCommits(t.Context(), nil)
		require.NoError(t, err)
		require.Len(t, commits, 3)
	})

	t.Run("RebaseOrMerge: rebases when safe", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		err = os.WriteFile(fmt.Sprintf("%s/local.txt", repo.Dir()), []byte("local"), 0o600)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "local commit", nil)
		require.NoError(t, err)

		err = internalWorkTree(t, repo).IntegrateRemoteChanges(
			t.Context(),
			&IntegrationOptions{
				TargetBranch:      "ahead",
				IntegrationPolicy: PushIntegrationPolicyRebaseOrMerge,
			},
		)
		require.NoError(t, err)

		// All local commits are unsigned and signing is not configured, so
		// rebase is safe. No merge commits: initial + remote + local = 3.
		commits, err := repo.ListCommits(t.Context(), nil)
		require.NoError(t, err)
		require.Len(t, commits, 3)
	})

	t.Run("RebaseOrMerge: merges when rebase is unsafe", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		err = os.WriteFile(fmt.Sprintf("%s/local.txt", repo.Dir()), []byte("local"), 0o600)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "local commit", nil)
		require.NoError(t, err)

		// Enable signing so that rebasing unsigned commits would be unsafe.
		wt := internalWorkTree(t, repo)
		enableFakeCommitSigning(t, wt, true)

		err = wt.IntegrateRemoteChanges(t.Context(), &IntegrationOptions{
			TargetBranch:      "ahead",
			IntegrationPolicy: PushIntegrationPolicyRebaseOrMerge,
		})
		require.NoError(t, err)

		// Should have fallen back to merge.
		msg, err := repo.CommitMessage(t.Context(), "HEAD")
		require.NoError(t, err)
		require.Contains(t, msg, "Merge")
	})

	t.Run("RebaseOrFail: rebases when safe", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		err = os.WriteFile(fmt.Sprintf("%s/local.txt", repo.Dir()), []byte("local"), 0o600)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "local commit", nil)
		require.NoError(t, err)

		err = internalWorkTree(t, repo).IntegrateRemoteChanges(
			t.Context(),
			&IntegrationOptions{
				TargetBranch:      "ahead",
				IntegrationPolicy: PushIntegrationPolicyRebaseOrFail,
			},
		)
		require.NoError(t, err)

		// Rebase is safe (unsigned, signing not configured).
		commits, err := repo.ListCommits(t.Context(), nil)
		require.NoError(t, err)
		require.Len(t, commits, 3)
	})

	t.Run("RebaseOrFail: fails when rebase is unsafe", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		err = os.WriteFile(fmt.Sprintf("%s/local.txt", repo.Dir()), []byte("local"), 0o600)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "local commit", nil)
		require.NoError(t, err)

		wt := internalWorkTree(t, repo)
		enableFakeCommitSigning(t, wt, true)

		err = wt.IntegrateRemoteChanges(t.Context(), &IntegrationOptions{
			TargetBranch:      "ahead",
			IntegrationPolicy: PushIntegrationPolicyRebaseOrFail,
		})
		require.ErrorIs(t, err, ErrRebaseUnsafe)
	})

	t.Run("AlwaysMerge: merges unconditionally", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		err = os.WriteFile(fmt.Sprintf("%s/local.txt", repo.Dir()), []byte("local"), 0o600)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "local commit", nil)
		require.NoError(t, err)

		err = internalWorkTree(t, repo).IntegrateRemoteChanges(
			t.Context(),
			&IntegrationOptions{
				TargetBranch:      "ahead",
				IntegrationPolicy: PushIntegrationPolicyAlwaysMerge,
			},
		)
		require.NoError(t, err)

		// Even though rebase would be safe (unsigned, no signing), the policy
		// says always merge.
		msg, err := repo.CommitMessage(t.Context(), "HEAD")
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
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		// Add a conflicting local commit.
		err = os.WriteFile(
			fmt.Sprintf("%s/remote.txt", repo.Dir()),
			[]byte("from-local"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "local conflict", nil)
		require.NoError(t, err)

		// With a conflict, the rebase should have failed.
		err = internalWorkTree(t, repo).pullRebase(t.Context(), "ahead")
		require.ErrorIs(t, err, ErrMergeConflict)
	})

	t.Run("rebases succeeds", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		// Add a non-conflicting local commit.
		err = os.WriteFile(
			fmt.Sprintf("%s/local.txt", repo.Dir()),
			[]byte("local"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "local commit", nil)
		require.NoError(t, err)

		// With no conflicts, the rebase should have succeeded.
		err = internalWorkTree(t, repo).pullRebase(t.Context(), "ahead")
		require.NoError(t, err)

		// The rebase should not have created any merge commits, so the commit count
		// should be: initial + remote + local = 3.
		commits, err := repo.ListCommits(t.Context(), nil)
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
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		// Add a conflicting local commit.
		err = os.WriteFile(
			fmt.Sprintf("%s/remote.txt", repo.Dir()),
			[]byte("from-local"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "local conflict", nil)
		require.NoError(t, err)

		// With a conflict, the merge should have failed.
		err = internalWorkTree(t, repo).pullMerge(t.Context(), "ahead")
		require.ErrorIs(t, err, ErrMergeConflict)
	})

	t.Run("merge succeeds", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		// Add a non-conflicting local commit.
		err = os.WriteFile(
			fmt.Sprintf("%s/local.txt", repo.Dir()),
			[]byte("local"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "local commit", nil)
		require.NoError(t, err)

		// With no conflicts, the merge should have succeeded.
		require.NoError(
			t,
			internalWorkTree(t, repo).pullMerge(t.Context(), "ahead"),
		)

		// After a merge, the head of the local branch should be a merge commit.
		msg, err := repo.CommitMessage(t.Context(), "HEAD")
		require.NoError(t, err)
		require.Contains(t, msg, "Merge")
	})
}

func Test_workTree_canSafelyRebase(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(t, initialMainCommit)
	defer testServer.Close()

	t.Run("no commits to replay", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		safe, err := internalWorkTree(t, repo).canSafelyRebase(t.Context(), "main")
		require.NoError(t, err)
		require.True(t, safe)
	})

	t.Run("unsigned local commits to replay; signing not configured", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		// Add an unsigned local commit.
		err = os.WriteFile(
			fmt.Sprintf("%s/file.txt", repo.Dir()),
			[]byte("data"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "local commit", nil)
		require.NoError(t, err)

		safe, err := internalWorkTree(t, repo).canSafelyRebase(t.Context(), "main")
		require.NoError(t, err)
		// This should be safe because rebasing won't lend Kargo's signature to
		// commits that were previously unsigned.
		require.True(t, safe)
	})

	t.Run("unsigned commits to replay; signing configured", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		require.NoError(t, os.WriteFile(
			fmt.Sprintf("%s/file.txt", repo.Dir()),
			[]byte("data"),
			0o600,
		))
		err = repo.AddAllAndCommit(t.Context(), "local commit", nil)
		require.NoError(t, err)

		// Enable signing without actually importing a key.
		wt := internalWorkTree(t, repo)
		_, err = libExec.Exec(wt.buildGitCommand(
			t.Context(), "config", "--global", "commit.gpgSign", "true",
		))
		require.NoError(t, err)

		safe, err := wt.canSafelyRebase(t.Context(), "main")
		require.NoError(t, err)
		// This should be unsafe because rebasing would lend Kargo's signature to
		// commits that were previously unsigned.
		require.False(t, safe)
	})

	t.Run("trusted, signed local commits to replay; signing configured", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		wt := internalWorkTree(t, repo)
		enableFakeCommitSigning(t, wt, true)

		err = os.WriteFile(
			fmt.Sprintf("%s/file.txt", repo.Dir()),
			[]byte("data"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "signed commit", nil)
		require.NoError(t, err)

		safe, err := wt.canSafelyRebase(t.Context(), "main")
		require.NoError(t, err)
		// This should be safe because Kargo trusted the signature on commits it
		// will re-sign during the rebase.
		require.True(t, safe)
	})

	t.Run("trusted, signed local commits to replay; signing not configured", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		wt := internalWorkTree(t, repo)
		enableFakeCommitSigning(t, wt, true)

		err = os.WriteFile(
			fmt.Sprintf("%s/file.txt", repo.Dir()),
			[]byte("data"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "signed commit", nil)
		require.NoError(t, err)

		disableFakeCommitSigning(t, wt)

		safe, err := wt.canSafelyRebase(t.Context(), "main")
		require.NoError(t, err)
		// This should be unsafe because rebasing would strip trusted signatures
		// from existing commits.
		require.False(t, safe)
	})

	t.Run("untrusted, signed commits", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		wt := internalWorkTree(t, repo)
		enableFakeCommitSigning(t, wt, false)

		err = os.WriteFile(
			fmt.Sprintf("%s/file.txt", repo.Dir()),
			[]byte("data"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "untrusted signed commit", nil)
		require.NoError(t, err)

		// Unsafe regardless — Kargo can't vouch for this commit.
		safe, err := wt.canSafelyRebase(t.Context(), "main")
		require.NoError(t, err)
		// This should be unsafe, no matter what. If signing were not configured,
		// rebasing would strip signatures from existing commits. Something
		// downstream might trust those signatures. If signing were configured,
		// rebasing would lend Kargo's signature to commits it did not, itself,
		// trust.
		require.False(t, safe)
	})

	t.Run("full clone, remote branch rewritten", func(t *testing.T) {
		wt := newWorkTreeWithRewrittenRemoteBranch(t, 0)
		safe, err := wt.canSafelyRebase(t.Context(), "feature")
		require.NoError(t, err)
		// This should be unsafe because the rewrite leaves all three original
		// commits to replay, two of them unsigned. With signing configured,
		// rebasing would lend Kargo's signature to those commits.
		require.False(t, safe)
	})

	t.Run("shallow clone, remote branch rewritten", func(t *testing.T) {
		wt := newWorkTreeWithRewrittenRemoteBranch(t, 1)
		safe, err := wt.canSafelyRebase(t.Context(), "feature")
		require.NoError(t, err)
		// This should be unsafe for the same reason as with a full clone. The
		// shallow history hides the unsigned commits, but the remote branch no
		// longer shares history with the local one, so the commits to replay
		// can't be trusted to be complete.
		require.False(t, safe)
	})

	t.Run("shallow clone, remote branch advanced", func(t *testing.T) {
		clientOpts := &ClientOptions{Credentials: &testRepoCreds}

		// This is a stand-in for whoever else is working on the remote. It
		// uses its own branch to avoid disturbing the other cases.
		author, err := Clone(t.Context(), testRepoURL, clientOpts, nil)
		require.NoError(t, err)
		defer func() { require.NoError(t, author.Close(t.Context())) }()
		require.NoError(
			t,
			author.CreateChildBranch(t.Context(), "shallow-advanced"),
		)
		pushOpts := &PushOptions{TargetBranch: "shallow-advanced"}
		require.NoError(t, author.Push(t.Context(), pushOpts))

		bare, err := CloneBare(
			t.Context(),
			testRepoURL,
			clientOpts,
			&BareCloneOptions{
				Depth:    1,
				Branches: []string{"shallow-advanced"},
			},
		)
		require.NoError(t, err)
		defer func() { require.NoError(t, bare.Close(t.Context())) }()
		checkout, err := bare.AddWorkTree(
			t.Context(),
			filepath.Join(bare.HomeDir(), "shallow-advanced"),
			&AddWorkTreeOptions{Ref: "shallow-advanced"},
		)
		require.NoError(t, err)
		defer func() { require.NoError(t, checkout.Close(t.Context())) }()
		wt, ok := checkout.(*workTree)
		require.True(t, ok, "expected *workTree, got %T", checkout)
		enableFakeCommitSigning(t, wt, true)

		// Add a signed local commit that has not been pushed.
		require.NoError(t, os.WriteFile(
			fmt.Sprintf("%s/local.txt", checkout.Dir()),
			[]byte("local"),
			0o600,
		))
		require.NoError(
			t,
			checkout.AddAllAndCommit(t.Context(), "local commit", nil),
		)

		// Advance the remote branch.
		require.NoError(t, os.WriteFile(
			fmt.Sprintf("%s/remote.txt", author.Dir()),
			[]byte("remote"),
			0o600,
		))
		require.NoError(
			t,
			author.AddAllAndCommit(t.Context(), "remote commit", nil),
		)
		require.NoError(t, author.Push(t.Context(), pushOpts))

		safe, err := wt.canSafelyRebase(t.Context(), "shallow-advanced")
		require.NoError(t, err)
		// This should be safe because the local commit is signed by a trusted
		// key and signing is configured, so Kargo can re-sign it. The remote
		// branch only moved forward, so the shallow boundary is not in the
		// range of commits to replay.
		require.True(t, safe)
	})
}

// newWorkTreeWithRewrittenRemoteBranch returns a work tree whose remote branch
// was force-pushed after the work tree was created.
//
// The remote's "feature" branch starts with three commits on top of main:
//
//   - unsigned-1 (unsigned)
//   - unsigned-2 (unsigned)
//   - signed (signed with a trusted key)
//
// The work tree is a bare clone of that branch with signing configured. The
// remote branch is then replaced with a single unsigned commit on top of main.
func newWorkTreeWithRewrittenRemoteBranch(
	t *testing.T,
	depth uint,
) *workTree {
	t.Helper()

	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(
		t,
		initialMainCommit,
	)
	t.Cleanup(testServer.Close)
	clientOpts := &ClientOptions{Credentials: &testRepoCreds}

	// This is a stand-in for whoever else is working on the remote.
	author, err := Clone(t.Context(), testRepoURL, clientOpts, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, author.Close(context.Background())) })
	authorWT := internalWorkTree(t, author)
	commit := func(name string) {
		t.Helper()
		require.NoError(t, os.WriteFile(
			fmt.Sprintf("%s/%s.txt", author.Dir(), name),
			[]byte(name),
			0o600,
		))
		require.NoError(t, author.AddAllAndCommit(t.Context(), name, nil))
	}

	require.NoError(t, author.CreateChildBranch(t.Context(), "feature"))
	commit("unsigned-1")
	commit("unsigned-2")
	enableFakeCommitSigning(t, authorWT, true)
	commit("signed")
	require.NoError(
		t,
		author.Push(t.Context(), &PushOptions{TargetBranch: "feature"}),
	)

	bare, err := CloneBare(
		t.Context(),
		testRepoURL,
		clientOpts,
		&BareCloneOptions{Depth: depth, Branches: []string{"feature"}},
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, bare.Close(context.Background())) })
	checkout, err := bare.AddWorkTree(
		t.Context(),
		filepath.Join(bare.HomeDir(), "feature"),
		&AddWorkTreeOptions{Ref: "feature"},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, checkout.Close(context.Background()))
	})
	wt, ok := checkout.(*workTree)
	require.True(t, ok, "expected *workTree, got %T", checkout)
	enableFakeCommitSigning(t, wt, true)

	// Rewrite the remote's feature branch so that none of its original commits
	// remain on it.
	disableFakeCommitSigning(t, authorWT)
	require.NoError(t, author.Checkout(t.Context(), "main"))
	require.NoError(t, author.CreateChildBranch(t.Context(), "rewritten"))
	commit("rewritten")
	require.NoError(t, author.Push(
		t.Context(),
		&PushOptions{TargetBranch: "feature", Force: true},
	))

	return wt
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
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		commits, err := internalWorkTree(t, repo).commitsToReplay(t.Context(), "main")
		require.NoError(t, err)
		require.Empty(t, commits)
	})

	t.Run("local branch is behind remote target branch", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		commits, err := internalWorkTree(t, repo).
			commitsToReplay(t.Context(), "ahead")
		require.NoError(t, err)
		require.Empty(t, commits)
	})

	t.Run("local branch is ahead of remote target branch", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		err = os.WriteFile(
			fmt.Sprintf("%s/a.txt", repo.Dir()),
			[]byte("a"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "commit a", nil)
		require.NoError(t, err)
		err = os.WriteFile(
			fmt.Sprintf("%s/b.txt", repo.Dir()),
			[]byte("b"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "commit b", nil)
		require.NoError(t, err)

		commits, err := internalWorkTree(t, repo).
			commitsToReplay(t.Context(), "main")
		require.NoError(t, err)
		// Two local commits should need to be replayed on main.
		require.Len(t, commits, 2)
	})

	t.Run("local branch and remote target branch have diverged", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer repo.Close(t.Context())

		err = os.WriteFile(
			fmt.Sprintf("%s/a.txt", repo.Dir()),
			[]byte("a"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "commit a", nil)
		require.NoError(t, err)
		err = os.WriteFile(
			fmt.Sprintf("%s/b.txt", repo.Dir()),
			[]byte("b"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "commit b", nil)
		require.NoError(t, err)

		commits, err := internalWorkTree(t, repo).
			commitsToReplay(t.Context(), "ahead")
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
