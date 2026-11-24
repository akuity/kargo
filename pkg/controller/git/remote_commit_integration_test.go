package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func Test_workTree_IntegrateRemoteChanges_ForkPoint(t *testing.T) {
	testCases := []struct {
		name           string
		localCommit    bool
		commitSigned   bool
		signingEnabled bool
		policy         PushIntegrationPolicy
		expectedErr    error
	}{
		{
			name:        "AlwaysRebase drops rewritten upstream commit",
			localCommit: true,
			policy:      PushIntegrationPolicyAlwaysRebase,
		},
		{
			name:           "RebaseOrFail inspects only commits after fork point",
			localCommit:    true,
			commitSigned:   true,
			signingEnabled: true,
			policy:         PushIntegrationPolicyRebaseOrFail,
		},
		{
			name:           "RebaseOrFail rejects unsigned local commit after fork point",
			localCommit:    true,
			signingEnabled: true,
			policy:         PushIntegrationPolicyRebaseOrFail,
			expectedErr:    ErrRebaseUnsafe,
		},
		{
			name:           "RebaseOrFail with no local commits drops rewritten upstream commit",
			signingEnabled: true,
			policy:         PushIntegrationPolicyRebaseOrFail,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			repoURL, clientOpts, author := setupRemoteWithAuthor(t)

			// Push an unsigned commit "dropped" on top of initial main.
			commitFile(t, author, "dropped")
			require.NoError(t, author.Push(t.Context(), nil))

			// Full clone at "dropped", so origin/main's reflog records "dropped".
			repo, err := Clone(t.Context(), repoURL, clientOpts, nil)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, repo.Close(context.Background())) })

			wt := internalWorkTree(t, repo)
			if testCase.commitSigned {
				enableFakeCommitSigning(t, wt, true)
			}
			if testCase.localCommit {
				commitFile(t, wt, "local")
			}
			if testCase.signingEnabled && !testCase.commitSigned {
				enableFakeCommitSigning(t, wt, true)
			}

			// Rewrite remote main to drop "dropped" and replace it with "replacement".
			_, err = libExec.Exec(author.buildGitCommand(
				t.Context(), "reset", "--hard", "HEAD~1",
			))
			require.NoError(t, err)
			commitFile(t, author, "replacement")
			require.NoError(t, author.Push(
				t.Context(),
				&PushOptions{TargetBranch: "main", Force: true},
			))

			err = wt.IntegrateRemoteChanges(t.Context(), &IntegrationOptions{
				TargetBranch:      "main",
				IntegrationPolicy: testCase.policy,
			})
			if testCase.expectedErr != nil {
				require.ErrorIs(t, err, testCase.expectedErr)
				return
			}
			require.NoError(t, err)

			// Expect initial + replacement (+ local if committed); "dropped" must not
			// be replayed.
			expectedCommits := 2
			if testCase.localCommit {
				expectedCommits++
			}
			commits, err := wt.ListCommits(t.Context(), nil)
			require.NoError(t, err)
			require.Len(t, commits, expectedCommits)

			require.FileExists(t, filepath.Join(wt.Dir(), "replacement.txt"))
			require.NoFileExists(t, filepath.Join(wt.Dir(), "dropped.txt"))
			if testCase.localCommit {
				require.FileExists(t, filepath.Join(wt.Dir(), "local.txt"))
			}
		})
	}
}

func Test_workTree_rebase(t *testing.T) {
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
		defer func() { require.NoError(t, repo.Close(t.Context())) }()

		// Add a conflicting local commit.
		err = os.WriteFile(
			fmt.Sprintf("%s/remote.txt", repo.Dir()),
			[]byte("from-local"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "local conflict", nil)
		require.NoError(t, err)

		wt := internalWorkTree(t, repo)
		require.NoError(t, wt.Fetch(t.Context(), &FetchOptions{Branch: "ahead"}))

		// With a conflict, the rebase should have failed.
		err = wt.rebase(t.Context(), "origin/ahead")
		require.ErrorIs(t, err, ErrMergeConflict)
	})

	t.Run("rebase fails with merge conflicts in a bare clone's work tree", func(t *testing.T) {
		wt := cloneBareWorkTree(
			t,
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
			"main",
		)

		// Add a conflicting local commit.
		require.NoError(t, os.WriteFile(
			filepath.Join(wt.Dir(), "remote.txt"),
			[]byte("from-local"),
			0o600,
		))
		require.NoError(t, wt.AddAllAndCommit(t.Context(), "local conflict", nil))
		require.NoError(t, wt.Fetch(t.Context(), &FetchOptions{Branch: "ahead"}))

		// git reports a linked work tree's rebase state at an absolute path
		err := wt.rebase(t.Context(), "origin/ahead")
		require.ErrorIs(t, err, ErrMergeConflict)
		rebasing, err := wt.IsRebasing(t.Context())
		require.NoError(t, err)
		require.True(t, rebasing)

		// Leave the work tree clean enough to be removed
		_, err = libExec.Exec(wt.buildGitCommand(t.Context(), "rebase", "--abort"))
		require.NoError(t, err)
	})

	t.Run("rebase succeeds", func(t *testing.T) {
		repo, err := Clone(
			t.Context(),
			testRepoURL,
			&ClientOptions{Credentials: &testRepoCreds},
			nil,
		)
		require.NoError(t, err)
		defer func() { require.NoError(t, repo.Close(t.Context())) }()

		// Add a non-conflicting local commit.
		err = os.WriteFile(
			fmt.Sprintf("%s/local.txt", repo.Dir()),
			[]byte("local"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "local commit", nil)
		require.NoError(t, err)

		wt := internalWorkTree(t, repo)
		require.NoError(t, wt.Fetch(t.Context(), &FetchOptions{Branch: "ahead"}))

		// With no conflicts, the rebase should have succeeded.
		err = wt.rebase(t.Context(), "origin/ahead")
		require.NoError(t, err)

		// The rebase should not have created any merge commits, so the commit count
		// should be: initial + remote + local = 3.
		commits, err := repo.ListCommits(t.Context(), nil)
		require.NoError(t, err)
		require.Len(t, commits, 3)
	})
}

func Test_workTree_IntegrateRemoteChanges_TargetTipAlreadyPresent(t *testing.T) {
	// main:    initial -> c0 -> c1 -> c2 -> c3
	// release: initial -> c0 -> c1
	repoURL, clientOpts, author := setupRemoteWithAuthor(t)
	commitFile(t, author, "c0")
	commitFile(t, author, "c1")
	require.NoError(t, author.Push(t.Context(), nil))
	require.NoError(
		t,
		author.Push(t.Context(), &PushOptions{TargetBranch: "release"}),
	)
	commitFile(t, author, "c2")
	commitFile(t, author, "c3")
	require.NoError(t, author.Push(t.Context(), nil))

	// A shallow clone of main alone holds c0 through c3, so it already has the
	// tip of release, but no ref for it.
	wt := cloneBareWorkTree(
		t,
		repoURL,
		clientOpts,
		&BareCloneOptions{Depth: 4, Refs: []string{"refs/heads/main"}},
		"main",
	)
	require.Equal(t, 4, commitCount(t, wt.baseRepo, "HEAD"))

	require.NoError(t, wt.IntegrateRemoteChanges(t.Context(), &IntegrationOptions{
		TargetBranch:      "release",
		IntegrationPolicy: PushIntegrationPolicyRebaseOrMerge,
	}))

	// Fetching the tip of release must not cut the history beneath it short
	require.Equal(t, 4, commitCount(t, wt.baseRepo, "HEAD"))
}

func Test_workTree_IntegrateRemoteChanges_CrissCross(t *testing.T) {
	// Each branch merges the other's side of a fork, so HEAD and main have two
	// merge bases, b and c, whose common ancestor is a:
	//
	//   a -- b ------- m1 -- l    feature
	//    \     \     /
	//     \      \ /
	//      \     / \
	//       \  /     \
	//        c ------- m2 -- r    main
	repoURL, clientOpts, author := setupRemoteWithAuthor(t)
	merge := func(branch string) {
		_, err := libExec.Exec(author.buildGitCommand(
			t.Context(), "merge", "--no-edit", "--no-ff", branch,
		))
		require.NoError(t, err)
	}
	commitFile(t, author, "a")
	require.NoError(t, author.Push(t.Context(), nil))
	require.NoError(t, author.CreateChildBranch(t.Context(), "side-b"))
	commitFile(t, author, "b")
	require.NoError(t, author.Checkout(t.Context(), "main"))
	require.NoError(t, author.CreateChildBranch(t.Context(), "side-c"))
	commitFile(t, author, "c")
	require.NoError(t, author.Checkout(t.Context(), "side-b"))
	require.NoError(t, author.CreateChildBranch(t.Context(), "feature"))
	merge("side-c")
	commitFile(t, author, "l")
	require.NoError(t, author.Push(t.Context(), nil))
	require.NoError(t, author.Checkout(t.Context(), "side-c"))
	merge("side-b")
	commitFile(t, author, "r")
	require.NoError(
		t,
		author.Push(t.Context(), &PushOptions{TargetBranch: "main", Force: true}),
	)

	testCases := []struct {
		name   string
		depth  uint
		assert func(*testing.T, *workTree, error)
	}{
		{
			// At depth 3, b and c are shallow boundaries hiding a. Neither lies on
			// only one side, so history counts as complete, but git cannot
			// reconcile the two merge bases without a.
			name:  "shallow clone hiding the merge bases' common ancestor",
			depth: 3,
			assert: func(t *testing.T, wt *workTree, err error) {
				require.ErrorIs(t, err, ErrShallowHistoryIncomplete)
				// The error must come from the merge base check, not from the
				// earlier check of the history on either side
				complete, err := wt.ensureCompleteIntegrationHistory(t.Context(), "main")
				require.NoError(t, err)
				require.True(t, complete)
				mergeBases := strings.Fields(gitOutput(
					t, wt.baseRepo, "merge-base", "--all", "HEAD", "refs/remotes/origin/main",
				))
				require.Len(t, mergeBases, 2)
			},
		},
		{
			name: "full clone",
			assert: func(t *testing.T, wt *workTree, err error) {
				require.NoError(t, err)
				msg, err := wt.CommitMessage(t.Context(), "HEAD")
				require.NoError(t, err)
				require.True(t, strings.HasPrefix(msg, "Merge branch 'main' of "))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			wt := cloneBareWorkTree(
				t,
				repoURL,
				clientOpts,
				&BareCloneOptions{
					Depth: testCase.depth,
					Refs:  []string{"refs/heads/feature", "refs/heads/main"},
				},
				"feature",
			)
			testCase.assert(
				t,
				wt,
				wt.IntegrateRemoteChanges(t.Context(), &IntegrationOptions{
					TargetBranch:      "main",
					IntegrationPolicy: PushIntegrationPolicyAlwaysMerge,
				}),
			)
		})
	}
}

func Test_workTree_IntegrateRemoteChanges_MergeAroundShallowBoundary(t *testing.T) {
	// feature merges y, which reaches x without passing through s. main reaches
	// x only through s, and later undoes x in r:
	//
	//   initial -- x -- s -- r -- r1 -- r2 -- r3 -- r4    main
	//               \    \
	//                \    c1 -- c2 -- c3 -- c4 -- m       feature
	//                 \                          /
	//                  y ------------------------
	repoURL, clientOpts, author := setupRemoteWithAuthor(t)
	run := func(args ...string) {
		_, err := libExec.Exec(author.buildGitCommand(t.Context(), args...))
		require.NoError(t, err)
	}
	commitFile(t, author, "x")
	xID, err := author.LastCommitID(t.Context())
	require.NoError(t, err)
	require.NoError(t, author.CreateChildBranch(t.Context(), "side"))
	commitFile(t, author, "y")
	require.NoError(t, author.Checkout(t.Context(), "main"))
	commitFile(t, author, "s")
	require.NoError(t, author.CreateChildBranch(t.Context(), "feature"))
	for _, name := range []string{"c1", "c2", "c3", "c4"} {
		commitFile(t, author, name)
	}
	run("merge", "--no-edit", "--no-ff", "side")
	require.NoError(t, author.Push(t.Context(), nil))
	require.NoError(t, author.Checkout(t.Context(), "main"))
	run("revert", "--no-edit", xID)
	for _, name := range []string{"r1", "r2", "r3", "r4"} {
		commitFile(t, author, name)
	}
	require.NoError(t, author.Push(t.Context(), nil))

	// requireUndoKept asserts that x, which main undid, has not come back
	requireUndoKept := func(t *testing.T, wt *workTree) {
		t.Helper()
		_, err := os.Stat(filepath.Join(wt.Dir(), "x.txt"))
		require.ErrorIs(t, err, os.ErrNotExist)
	}
	requireMerged := func(t *testing.T, wt *workTree, err error) {
		t.Helper()
		require.NoError(t, err)
		// rev-list --parents lists HEAD, then its parents
		commits := strings.Fields(gitOutput(
			t, wt.baseRepo, "rev-list", "--parents", "-n", "1", "HEAD",
		))
		require.Len(t, commits, 3)
		require.Equal(
			t,
			revParse(t, wt.baseRepo, "refs/remotes/origin/main"),
			commits[2],
		)
		requireUndoKept(t, wt)
	}
	// At depth 6, s is the only shallow boundary. It is common to both
	// branches, so history counts as complete, yet it hides x from main.
	requireMisjudgedHistory := func(t *testing.T, wt *workTree) {
		t.Helper()
		shallow, err := wt.shallowCommits(t.Context())
		require.NoError(t, err)
		require.Len(t, shallow, 1)
		complete, err := wt.ensureCompleteIntegrationHistory(t.Context(), "main")
		require.NoError(t, err)
		require.True(t, complete)
		mayReplayShared, err := wt.mayReplaySharedCommits(
			t.Context(),
			"refs/remotes/origin/main",
		)
		require.NoError(t, err)
		require.True(t, mayReplayShared)
	}

	testCases := []struct {
		name   string
		depth  uint
		policy PushIntegrationPolicy
		assert func(*testing.T, *workTree, error)
	}{
		{
			name:   "shallow clone: AlwaysRebase fails on incomplete history",
			depth:  6,
			policy: PushIntegrationPolicyAlwaysRebase,
			assert: func(t *testing.T, wt *workTree, err error) {
				require.ErrorIs(t, err, ErrShallowHistoryIncomplete)
				requireMisjudgedHistory(t, wt)
			},
		},
		{
			name:   "shallow clone: RebaseOrFail returns ErrRebaseUnsafe",
			depth:  6,
			policy: PushIntegrationPolicyRebaseOrFail,
			assert: func(t *testing.T, wt *workTree, err error) {
				require.ErrorIs(t, err, ErrRebaseUnsafe)
				requireMisjudgedHistory(t, wt)
			},
		},
		{
			name:   "shallow clone: RebaseOrMerge merges",
			depth:  6,
			policy: PushIntegrationPolicyRebaseOrMerge,
			assert: requireMerged,
		},
		{
			name:   "shallow clone: AlwaysMerge merges",
			depth:  6,
			policy: PushIntegrationPolicyAlwaysMerge,
			assert: requireMerged,
		},
		{
			name:   "full clone: RebaseOrFail rebases",
			policy: PushIntegrationPolicyRebaseOrFail,
			assert: func(t *testing.T, wt *workTree, err error) {
				require.NoError(t, err)
				_, err = libExec.Exec(wt.buildGitCommand(
					t.Context(),
					"merge-base", "--is-ancestor", "refs/remotes/origin/main", "HEAD",
				))
				require.NoError(t, err)
				require.Empty(t, gitOutput(
					t, wt.baseRepo,
					"rev-list", "--merges", "HEAD", "^refs/remotes/origin/main", "--",
				))
				requireUndoKept(t, wt)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			wt := cloneBareWorkTree(
				t,
				repoURL,
				clientOpts,
				&BareCloneOptions{
					Depth: testCase.depth,
					Refs:  []string{"refs/heads/feature", "refs/heads/main"},
				},
				"feature",
			)
			commitFile(t, wt, "local")
			headBefore, err := wt.LastCommitID(t.Context())
			require.NoError(t, err)
			err = wt.IntegrateRemoteChanges(t.Context(), &IntegrationOptions{
				TargetBranch:      "main",
				IntegrationPolicy: testCase.policy,
			})
			testCase.assert(t, wt, err)
			if err != nil {
				headAfter, headErr := wt.LastCommitID(t.Context())
				require.NoError(t, headErr)
				require.Equal(t, headBefore, headAfter)
			}
		})
	}
}

func Test_workTree_IntegrateRemoteChanges_ShallowOutsideTarget(t *testing.T) {
	// main:    initial -- a -- r
	// feature: a -- f -- m, where m merges a -- g
	// archive: initial -- o1 -- o2 -- o3 -- o4 -- o5 -- o6
	//
	// At depth 5, main and feature are complete, and only archive's history is
	// cut short.
	repoURL, clientOpts, author := setupRemoteWithAuthor(t)
	run := func(args ...string) {
		_, err := libExec.Exec(author.buildGitCommand(t.Context(), args...))
		require.NoError(t, err)
	}
	commitFile(t, author, "a")
	require.NoError(t, author.CreateChildBranch(t.Context(), "side"))
	commitFile(t, author, "g")
	require.NoError(t, author.Checkout(t.Context(), "main"))
	require.NoError(t, author.CreateChildBranch(t.Context(), "feature"))
	commitFile(t, author, "f")
	run("merge", "--no-edit", "--no-ff", "side")
	require.NoError(t, author.Push(t.Context(), nil))
	require.NoError(t, author.Checkout(t.Context(), "main"))
	run("checkout", "-b", "archive", "HEAD~1")
	for _, name := range []string{"o1", "o2", "o3", "o4", "o5", "o6"} {
		commitFile(t, author, name)
	}
	require.NoError(t, author.Push(t.Context(), nil))
	require.NoError(t, author.Checkout(t.Context(), "main"))
	commitFile(t, author, "r")
	require.NoError(t, author.Push(t.Context(), nil))

	testCases := []struct {
		name   string
		policy PushIntegrationPolicy
	}{
		{
			name:   "AlwaysRebase rebases",
			policy: PushIntegrationPolicyAlwaysRebase,
		},
		{
			name:   "RebaseOrFail rebases",
			policy: PushIntegrationPolicyRebaseOrFail,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			wt := cloneBareWorkTree(
				t,
				repoURL,
				clientOpts,
				&BareCloneOptions{
					Depth: 5,
					Refs: []string{
						"refs/heads/feature",
						"refs/heads/main",
						"refs/heads/archive",
					},
				},
				"feature",
			)
			commitFile(t, wt, "local")
			shallow, err := wt.shallowCommits(t.Context())
			require.NoError(t, err)
			require.Equal(
				t,
				map[string]struct{}{
					revParse(t, wt.baseRepo, "refs/heads/archive~4"): {},
				},
				shallow,
			)

			err = wt.IntegrateRemoteChanges(t.Context(), &IntegrationOptions{
				TargetBranch:      "main",
				IntegrationPolicy: testCase.policy,
			})
			require.NoError(t, err)
			_, err = libExec.Exec(wt.buildGitCommand(
				t.Context(),
				"merge-base", "--is-ancestor", "refs/remotes/origin/main", "HEAD",
			))
			require.NoError(t, err)
			require.Empty(t, gitOutput(
				t, wt.baseRepo,
				"rev-list", "--merges", "HEAD", "^refs/remotes/origin/main", "--",
			))
			require.Equal(t, "local", subject(t, wt.baseRepo, "HEAD"))
		})
	}
}

func Test_workTree_merge(t *testing.T) {
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
		defer func() { require.NoError(t, repo.Close(t.Context())) }()

		// Add a conflicting local commit.
		err = os.WriteFile(
			fmt.Sprintf("%s/remote.txt", repo.Dir()),
			[]byte("from-local"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "local conflict", nil)
		require.NoError(t, err)

		wt := internalWorkTree(t, repo)
		require.NoError(t, wt.Fetch(t.Context(), &FetchOptions{Branch: "ahead"}))

		// With a conflict, the merge should have failed.
		err = wt.merge(t.Context(), "ahead")
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
		defer func() { require.NoError(t, repo.Close(t.Context())) }()

		// Add a non-conflicting local commit.
		err = os.WriteFile(
			fmt.Sprintf("%s/local.txt", repo.Dir()),
			[]byte("local"),
			0o600,
		)
		require.NoError(t, err)
		err = repo.AddAllAndCommit(t.Context(), "local commit", nil)
		require.NoError(t, err)

		wt := internalWorkTree(t, repo)
		require.NoError(t, wt.Fetch(t.Context(), &FetchOptions{Branch: "ahead"}))

		// With no conflicts, the merge should have succeeded.
		require.NoError(t, wt.merge(t.Context(), "ahead"))

		// After a merge, the head of the local branch should be a merge commit
		// with the message git pull would have given it. Like git, it omits the
		// URL's .git suffix and, when merging into main, the destination branch.
		msg, err := repo.CommitMessage(t.Context(), "HEAD")
		require.NoError(t, err)
		require.Equal(
			t,
			"Merge branch 'ahead' of "+strings.TrimSuffix(testRepoURL, ".git"),
			strings.TrimSpace(msg),
		)
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

		wt := internalWorkTree(t, repo)
		require.NoError(t, wt.Fetch(t.Context(), &FetchOptions{Branch: "main"}))
		safe, err := wt.canSafelyRebase(t.Context(), "origin/main")
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

		wt := internalWorkTree(t, repo)
		require.NoError(t, wt.Fetch(t.Context(), &FetchOptions{Branch: "main"}))
		safe, err := wt.canSafelyRebase(t.Context(), "origin/main")
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

		require.NoError(t, wt.Fetch(t.Context(), &FetchOptions{Branch: "main"}))
		safe, err := wt.canSafelyRebase(t.Context(), "origin/main")
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

		require.NoError(t, wt.Fetch(t.Context(), &FetchOptions{Branch: "main"}))
		safe, err := wt.canSafelyRebase(t.Context(), "origin/main")
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

		require.NoError(t, wt.Fetch(t.Context(), &FetchOptions{Branch: "main"}))
		safe, err := wt.canSafelyRebase(t.Context(), "origin/main")
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
		require.NoError(t, wt.Fetch(t.Context(), &FetchOptions{Branch: "main"}))
		safe, err := wt.canSafelyRebase(t.Context(), "origin/main")
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
		require.NoError(t, wt.Fetch(t.Context(), &FetchOptions{Branch: "feature"}))
		safe, err := wt.canSafelyRebase(t.Context(), "origin/feature")
		require.NoError(t, err)
		// This should be unsafe because the rewrite leaves all three original
		// commits to replay, two of them unsigned. With signing configured,
		// rebasing would lend Kargo's signature to those commits.
		require.False(t, safe)
	})

	t.Run("shallow clone, remote branch rewritten", func(t *testing.T) {
		wt := newWorkTreeWithRewrittenRemoteBranch(t, 1)
		require.NoError(t, wt.Fetch(t.Context(), &FetchOptions{Branch: "feature"}))
		safe, err := wt.canSafelyRebase(t.Context(), "origin/feature")
		require.NoError(t, err)
		// This should be unsafe for the same reason as with a full clone. The
		// shallow history hides the unsigned commits, so a shallow boundary
		// remains in the symmetric difference and the commits to replay cannot
		// be trusted to be complete.
		require.False(t, safe)
	})

	t.Run("shallow clone, remote branch advanced", func(t *testing.T) {
		clientOpts := &ClientOptions{Credentials: &testRepoCreds}

		// This is a stand-in for whoever else is working on the remote. It
		// uses its own branch to avoid disturbing the other cases.
		author, err := Clone(t.Context(), testRepoURL, clientOpts, nil)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, author.Close(context.Background())) })
		require.NoError(
			t,
			author.CreateChildBranch(t.Context(), "shallow-advanced"),
		)
		pushOpts := &PushOptions{TargetBranch: "shallow-advanced"}
		require.NoError(t, author.Push(t.Context(), pushOpts))

		wt := cloneBareWorkTree(
			t,
			testRepoURL,
			clientOpts,
			&BareCloneOptions{Depth: 1, Refs: []string{"refs/heads/shallow-advanced"}},
			"shallow-advanced",
		)
		enableFakeCommitSigning(t, wt, true)

		// Add a signed local commit and advance the remote branch.
		commitFile(t, wt, "local")
		commitFile(t, author, "remote")
		require.NoError(t, author.Push(t.Context(), pushOpts))

		require.NoError(
			t,
			wt.Fetch(t.Context(), &FetchOptions{Branch: "shallow-advanced"}),
		)
		safe, err := wt.canSafelyRebase(t.Context(), "origin/shallow-advanced")
		require.NoError(t, err)
		// This should be safe because the local commit is signed by a trusted
		// key and signing is configured, so Kargo can re-sign it. The remote
		// branch only moved forward, so the shallow boundary is not in the
		// symmetric difference.
		require.True(t, safe)
	})
}

// setupRemoteWithAuthor creates an in-process remote repository initialized
// with an initial commit on main and returns its URL, client options, and a
// full-clone work tree acting as another collaborator pushing to the remote.
func setupRemoteWithAuthor(
	t *testing.T,
) (string, *ClientOptions, *workTree) {
	t.Helper()

	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(
		t,
		initialMainCommit,
	)
	t.Cleanup(testServer.Close)
	clientOpts := &ClientOptions{Credentials: &testRepoCreds}

	author, err := Clone(t.Context(), testRepoURL, clientOpts, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, author.Close(context.Background())) })

	return testRepoURL, clientOpts, internalWorkTree(t, author)
}

// commitFile writes "<name>.txt" to wt and commits it with message "<name>".
func commitFile(t *testing.T, wt WorkTree, name string) {
	t.Helper()
	require.NoError(t, os.WriteFile(
		filepath.Join(wt.Dir(), name+".txt"),
		[]byte(name),
		0o600,
	))
	require.NoError(t, wt.AddAllAndCommit(t.Context(), name, nil))
}

// cloneBareWorkTree performs a bare clone and adds a working tree for ref,
// registering cleanups for both.
func cloneBareWorkTree(
	t *testing.T,
	repoURL string,
	clientOpts *ClientOptions,
	cloneOpts *BareCloneOptions,
	ref string,
) *workTree {
	t.Helper()
	bare, err := CloneBare(t.Context(), repoURL, clientOpts, cloneOpts)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, bare.Close(context.Background())) })

	checkout, err := bare.AddWorkTree(
		t.Context(),
		filepath.Join(bare.HomeDir(), ref),
		&AddWorkTreeOptions{Ref: ref},
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, checkout.Close(context.Background()))
	})
	wt, ok := checkout.(*workTree)
	require.True(t, ok, "expected *workTree, got %T", checkout)
	return wt
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

	repoURL, clientOpts, author := setupRemoteWithAuthor(t)

	require.NoError(t, author.CreateChildBranch(t.Context(), "feature"))
	commitFile(t, author, "unsigned-1")
	commitFile(t, author, "unsigned-2")
	enableFakeCommitSigning(t, author, true)
	commitFile(t, author, "signed")
	require.NoError(
		t,
		author.Push(t.Context(), &PushOptions{TargetBranch: "feature"}),
	)

	wt := cloneBareWorkTree(
		t,
		repoURL,
		clientOpts,
		&BareCloneOptions{Depth: depth, Refs: []string{"refs/heads/feature"}},
		"feature",
	)
	enableFakeCommitSigning(t, wt, true)

	// Rewrite the remote's feature branch so that none of its original commits
	// remain on it.
	disableFakeCommitSigning(t, author)
	require.NoError(t, author.Checkout(t.Context(), "main"))
	require.NoError(t, author.CreateChildBranch(t.Context(), "rewritten"))
	commitFile(t, author, "rewritten")
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

		wt := internalWorkTree(t, repo)
		require.NoError(t, wt.Fetch(t.Context(), &FetchOptions{Branch: "main"}))
		commits, err := wt.commitsToReplay(t.Context(), "origin/main")
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

		wt := internalWorkTree(t, repo)
		require.NoError(t, wt.Fetch(t.Context(), &FetchOptions{Branch: "ahead"}))
		commits, err := wt.commitsToReplay(t.Context(), "origin/ahead")
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

		wt := internalWorkTree(t, repo)
		require.NoError(t, wt.Fetch(t.Context(), &FetchOptions{Branch: "main"}))
		commits, err := wt.commitsToReplay(t.Context(), "origin/main")
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

		wt := internalWorkTree(t, repo)
		require.NoError(t, wt.Fetch(t.Context(), &FetchOptions{Branch: "ahead"}))
		commits, err := wt.commitsToReplay(t.Context(), "origin/ahead")
		// Two local commits should need to be replayed on ahead.
		require.NoError(t, err)
		require.Len(t, commits, 2)
	})
}

func Test_workTree_IntegrateRemoteChanges_ShallowClone(t *testing.T) {
	// Set up a single remote repo shared across the table cases:
	//   - main:             initial -> base -> remote-1 -> remote-2
	//   - feature-unsigned: base -> commit-1 (unsigned) -> commit-2 (unsigned) -> signed-tip (signed)
	//   - feature-signed:   base -> signed-1 (signed)   -> signed-2 (signed)   -> signed-tip (signed)
	repoURL, clientOpts, author := setupRemoteWithAuthor(t)
	commitFile(t, author, "base")
	require.NoError(t, author.Push(t.Context(), nil))

	require.NoError(t, author.CreateChildBranch(t.Context(), "feature-unsigned"))
	commitFile(t, author, "commit-1")
	commitFile(t, author, "commit-2")
	enableFakeCommitSigning(t, author, true)
	commitFile(t, author, "signed-tip")
	require.NoError(
		t,
		author.Push(t.Context(), &PushOptions{TargetBranch: "feature-unsigned"}),
	)

	require.NoError(t, author.Checkout(t.Context(), "main"))
	require.NoError(t, author.CreateChildBranch(t.Context(), "feature-signed"))
	commitFile(t, author, "signed-1")
	commitFile(t, author, "signed-2")
	commitFile(t, author, "signed-tip")
	require.NoError(
		t,
		author.Push(t.Context(), &PushOptions{TargetBranch: "feature-signed"}),
	)

	disableFakeCommitSigning(t, author)
	require.NoError(t, author.Checkout(t.Context(), "main"))
	commitFile(t, author, "remote-1")
	commitFile(t, author, "remote-2")
	require.NoError(t, author.Push(t.Context(), nil))

	cloneShallowBranch := func(t *testing.T, branch string) *workTree {
		t.Helper()
		wt := cloneBareWorkTree(
			t,
			repoURL,
			clientOpts,
			&BareCloneOptions{Depth: 1, Refs: []string{"refs/heads/" + branch}},
			branch,
		)
		enableFakeCommitSigning(t, wt, true)
		return wt
	}

	// Creates a local branch at base (shared with main) and merges a depth-1
	// fetch of feature-unsigned so HEAD is a signed merge commit with a valid
	// merge-base to main, yet its 2nd parent is shallow and hides commit-1/2.
	setupMergeWithShallowSideParent := func(t *testing.T) *workTree {
		t.Helper()
		wt := cloneBareWorkTree(
			t,
			repoURL,
			clientOpts,
			&BareCloneOptions{Depth: 3, Refs: []string{"refs/heads/main"}},
			"main",
		)
		_, err := libExec.Exec(wt.buildGitCommand(
			t.Context(), "checkout", "-b", "feat", "HEAD~2",
		))
		require.NoError(t, err)
		require.NoError(
			t,
			wt.Fetch(
				t.Context(),
				&FetchOptions{Branch: "feature-unsigned", Depth: 1},
			),
		)
		enableFakeCommitSigning(t, wt, true)
		_, err = libExec.Exec(wt.buildGitCommand(
			t.Context(),
			"merge",
			"--no-edit",
			"--allow-unrelated-histories",
			"origin/feature-unsigned",
		))
		require.NoError(t, err)
		return wt
	}

	// Clones main at depth 1 (shallow target) while fetching feature-signed in
	// full (complete HEAD). Without deepening main, origin/main..HEAD would
	// falsely include the unsigned shared base commit.
	setupTargetShallowCompleteHead := func(t *testing.T) *workTree {
		t.Helper()
		wt := cloneBareWorkTree(
			t,
			repoURL,
			clientOpts,
			&BareCloneOptions{Depth: 1, Refs: []string{"refs/heads/main"}},
			"main",
		)
		require.NoError(
			t,
			wt.Fetch(t.Context(), &FetchOptions{Branch: "feature-signed"}),
		)
		_, err := libExec.Exec(wt.buildGitCommand(
			t.Context(), "checkout", "-b", "feature-signed", "origin/feature-signed",
		))
		require.NoError(t, err)
		enableFakeCommitSigning(t, wt, true)
		return wt
	}

	assertRebased := func(
		t *testing.T,
		wt *workTree,
		err error,
		expectedCount int,
		expectedLeadingSubjects ...string,
	) {
		t.Helper()
		require.NoError(t, err)
		commits, listErr := wt.ListCommits(t.Context(), nil)
		require.NoError(t, listErr)
		require.Len(t, commits, expectedCount)
		for i, subject := range expectedLeadingSubjects {
			require.Equal(t, subject, commits[i].Subject)
		}
	}

	// assertMerged asserts that HEAD is a new merge commit that merged main into
	// the current branch, with the message git pull would have given it.
	assertMerged := func(t *testing.T, wt *workTree, err error) {
		t.Helper()
		require.NoError(t, err)
		// rev-list --parents lists HEAD, then its parents
		commits := strings.Fields(gitOutput(
			t, wt.baseRepo, "rev-list", "--parents", "-n", "1", "HEAD",
		))
		require.Len(t, commits, 3)
		require.Equal(t, revParse(t, wt.baseRepo, "refs/remotes/origin/main"), commits[2])
		currentBranch, branchErr := wt.CurrentBranch(t.Context())
		require.NoError(t, branchErr)
		msg, msgErr := wt.CommitMessage(t.Context(), "HEAD")
		require.NoError(t, msgErr)
		require.Equal(
			t,
			fmt.Sprintf(
				"Merge branch 'main' of %s into %s",
				strings.TrimSuffix(repoURL, ".git"), currentBranch,
			),
			strings.TrimSpace(msg),
		)
	}

	testCases := []struct {
		name         string
		setup        func(t *testing.T) *workTree
		targetBranch string
		policy       PushIntegrationPolicy
		assert       func(t *testing.T, wt *workTree, err error)
	}{
		{
			name: "diverged past depth: AlwaysRebase deepens and preserves all commits",
			setup: func(t *testing.T) *workTree {
				return cloneShallowBranch(t, "feature-unsigned")
			},
			targetBranch: "main",
			policy:       PushIntegrationPolicyAlwaysRebase,
			assert: func(t *testing.T, wt *workTree, err error) {
				assertRebased(
					t, wt, err,
					7, "signed-tip", "commit-2", "commit-1", "remote-2",
				)
			},
		},
		{
			name: "diverged past depth with hidden unsigned commits: RebaseOrFail returns ErrRebaseUnsafe",
			setup: func(t *testing.T) *workTree {
				return cloneShallowBranch(t, "feature-unsigned")
			},
			targetBranch: "main",
			policy:       PushIntegrationPolicyRebaseOrFail,
			assert: func(t *testing.T, _ *workTree, err error) {
				require.ErrorIs(t, err, ErrRebaseUnsafe)
			},
		},
		{
			name: "diverged past depth with hidden unsigned commits: RebaseOrMerge deepens and merges cleanly",
			setup: func(t *testing.T) *workTree {
				return cloneShallowBranch(t, "feature-unsigned")
			},
			targetBranch: "main",
			policy:       PushIntegrationPolicyRebaseOrMerge,
			assert: func(t *testing.T, wt *workTree, err error) {
				assertMerged(t, wt, err)
			},
		},
		{
			name: "diverged past depth: AlwaysMerge deepens and merges cleanly",
			setup: func(t *testing.T) *workTree {
				return cloneShallowBranch(t, "feature-unsigned")
			},
			targetBranch: "main",
			policy:       PushIntegrationPolicyAlwaysMerge,
			assert: func(t *testing.T, wt *workTree, err error) {
				assertMerged(t, wt, err)
			},
		},
		{
			name: "diverged past depth with all signed commits: RebaseOrFail deepens and rebases cleanly",
			setup: func(t *testing.T) *workTree {
				return cloneShallowBranch(t, "feature-signed")
			},
			targetBranch: "main",
			policy:       PushIntegrationPolicyRebaseOrFail,
			assert: func(t *testing.T, wt *workTree, err error) {
				assertRebased(
					t, wt, err,
					7, "signed-tip", "signed-2", "signed-1", "remote-2",
				)
			},
		},
		{
			name: "target branch shallow while HEAD is complete: deepens target and rebases safely",
			setup: func(t *testing.T) *workTree {
				return setupTargetShallowCompleteHead(t)
			},
			targetBranch: "main",
			policy:       PushIntegrationPolicyRebaseOrFail,
			assert: func(t *testing.T, wt *workTree, err error) {
				assertRebased(
					t, wt, err,
					7, "signed-tip", "signed-2", "signed-1", "remote-2",
				)
			},
		},
		// Whether deepening main also recovers the history behind the shallow
		// side parent depends on the version of git. If it does, that history
		// holds unsigned commits, so rebasing is unsafe. If it does not, the
		// history is incomplete. Either way, there must be no rebase.
		{
			name: "merge commit with shallow side parent: RebaseOrFail does not rebase",
			setup: func(t *testing.T) *workTree {
				return setupMergeWithShallowSideParent(t)
			},
			targetBranch: "main",
			policy:       PushIntegrationPolicyRebaseOrFail,
			assert: func(t *testing.T, _ *workTree, err error) {
				require.True(
					t,
					errors.Is(err, ErrRebaseUnsafe) ||
						errors.Is(err, ErrShallowHistoryIncomplete),
					"unexpected error: %v", err,
				)
			},
		},
		{
			name: "merge commit with shallow side parent: RebaseOrMerge does not rebase",
			setup: func(t *testing.T) *workTree {
				return setupMergeWithShallowSideParent(t)
			},
			targetBranch: "main",
			policy:       PushIntegrationPolicyRebaseOrMerge,
			assert: func(t *testing.T, wt *workTree, err error) {
				if errors.Is(err, ErrShallowHistoryIncomplete) {
					return
				}
				assertMerged(t, wt, err)
			},
		},
		{
			name: "remote branch rewritten: AlwaysRebase fails on incomplete history",
			setup: func(t *testing.T) *workTree {
				return newWorkTreeWithRewrittenRemoteBranch(t, 1)
			},
			targetBranch: "feature",
			policy:       PushIntegrationPolicyAlwaysRebase,
			assert: func(t *testing.T, _ *workTree, err error) {
				require.ErrorIs(t, err, ErrShallowHistoryIncomplete)
			},
		},
		{
			name: "remote branch rewritten: AlwaysMerge fails on incomplete history",
			setup: func(t *testing.T) *workTree {
				return newWorkTreeWithRewrittenRemoteBranch(t, 1)
			},
			targetBranch: "feature",
			policy:       PushIntegrationPolicyAlwaysMerge,
			assert: func(t *testing.T, _ *workTree, err error) {
				require.ErrorIs(t, err, ErrShallowHistoryIncomplete)
			},
		},
		{
			name: "remote branch rewritten: RebaseOrFail fails on incomplete history",
			setup: func(t *testing.T) *workTree {
				return newWorkTreeWithRewrittenRemoteBranch(t, 1)
			},
			targetBranch: "feature",
			policy:       PushIntegrationPolicyRebaseOrFail,
			assert: func(t *testing.T, _ *workTree, err error) {
				require.ErrorIs(t, err, ErrShallowHistoryIncomplete)
			},
		},
		{
			name: "remote branch rewritten: RebaseOrMerge fails on incomplete history",
			setup: func(t *testing.T) *workTree {
				return newWorkTreeWithRewrittenRemoteBranch(t, 1)
			},
			targetBranch: "feature",
			policy:       PushIntegrationPolicyRebaseOrMerge,
			assert: func(t *testing.T, _ *workTree, err error) {
				require.ErrorIs(t, err, ErrShallowHistoryIncomplete)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			wt := testCase.setup(t)
			headBefore, err := wt.LastCommitID(t.Context())
			require.NoError(t, err)
			err = wt.IntegrateRemoteChanges(t.Context(), &IntegrationOptions{
				TargetBranch:      testCase.targetBranch,
				IntegrationPolicy: testCase.policy,
			})
			testCase.assert(t, wt, err)
			if err != nil {
				headAfter, headErr := wt.LastCommitID(t.Context())
				require.NoError(t, headErr)
				require.Equal(t, headBefore, headAfter)
				rebasing, rbErr := wt.IsRebasing(t.Context())
				require.NoError(t, rbErr)
				require.False(t, rebasing)
			}
		})
	}

	t.Run("fast path: shallow clone with remote moving forward does not deepen", func(t *testing.T) {
		require.NoError(t, author.CreateChildBranch(t.Context(), "fast-path"))
		pushOpts := &PushOptions{TargetBranch: "fast-path"}
		require.NoError(t, author.Push(t.Context(), pushOpts))

		wt := cloneShallowBranch(t, "fast-path")

		initialShallow, err := wt.shallowCommits(t.Context())
		require.NoError(t, err)
		require.Len(t, initialShallow, 1)

		// Add one signed local commit and one remote commit from the same tip.
		commitFile(t, wt, "local-signed")
		commitFile(t, author, "remote-new")
		require.NoError(t, author.Push(t.Context(), pushOpts))

		err = wt.IntegrateRemoteChanges(t.Context(), &IntegrationOptions{
			TargetBranch:      "fast-path",
			IntegrationPolicy: PushIntegrationPolicyRebaseOrFail,
		})
		assertRebased(t, wt, err, 3, "local-signed", "remote-new")

		// Shallow boundary should be completely unchanged (0 deepening fetches).
		afterShallow, err := wt.shallowCommits(t.Context())
		require.NoError(t, err)
		require.Equal(t, initialShallow, afterShallow)
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
