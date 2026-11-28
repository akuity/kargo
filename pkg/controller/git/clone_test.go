package git

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	libExec "github.com/akuity/kargo/pkg/exec"
)

func TestClone(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(
		t,
		initMultipleBranches,
	)
	t.Cleanup(testServer.Close)
	clientOpts := &ClientOptions{Credentials: &testRepoCreds}
	branchTip, unservedCommit, taggedCommit := testCommitIDs(
		t,
		testRepoURL,
		clientOpts,
	)

	testCases := []struct {
		name   string
		opts   *CloneOptions
		assert func(*testing.T, *repo, error)
	}{
		{
			name: "unsupported ref",
			opts: &CloneOptions{Refs: []string{"main"}},
			assert: func(t *testing.T, _ *repo, err error) {
				require.ErrorContains(t, err, `unsupported ref "main"`)
			},
		},
		{
			name: "unsupported checkout",
			opts: &CloneOptions{Checkout: "main"},
			assert: func(t *testing.T, _ *repo, err error) {
				require.ErrorContains(t, err, `unsupported ref "main"`)
			},
		},
		{
			name: "no refs clones all branches and tags",
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				requireRefs(t, r.baseRepo, remoteBranchRefPrefix, "a", "b", "main")
				requireRefs(t, r.baseRepo, tagRefPrefix, "b", "v1.0.0")
				requireCurrentBranch(t, r, "main")
				require.False(t, isShallowRepo(t, r.baseRepo))
			},
		},
		{
			name: "HEAD clones only the default branch",
			opts: &CloneOptions{Refs: []string{headRef}},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				requireRefs(t, r.baseRepo, remoteBranchRefPrefix, "main")
				requireRefs(t, r.baseRepo, tagRefPrefix)
				requireCurrentBranch(t, r, "main")
			},
		},
		{
			name: "what is checked out is cloned whether or not refs include it",
			opts: &CloneOptions{
				Refs:     []string{"refs/heads/a"},
				Checkout: "refs/heads/b",
			},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				requireRefs(t, r.baseRepo, remoteBranchRefPrefix, "a", "b")
				requireCurrentBranch(t, r, "b")
			},
		},
		{
			name: "refs missing from the remote are skipped",
			opts: &CloneOptions{
				Refs: []string{"refs/heads/missing", "refs/heads/a"},
			},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				requireRefs(t, r.baseRepo, remoteBranchRefPrefix, "a", "main")
				requireCurrentBranch(t, r, "main")
			},
		},
		{
			name: "branches that are not checked out can be checked out later",
			opts: &CloneOptions{
				Refs: []string{headRef, "refs/heads/a"},
			},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				require.NoError(t, r.Checkout(t.Context(), "a"))
				requireCurrentBranch(t, r, "a")
			},
		},
		{
			name: "branches can be checked out later after checking out a tag",
			opts: &CloneOptions{
				Refs:     []string{"refs/heads/a"},
				Checkout: "refs/tags/v1.0.0",
			},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				requireCurrentBranch(t, r, "")
				require.NoError(t, r.Checkout(t.Context(), "a"))
				requireCurrentBranch(t, r, "a")
			},
		},
		{
			name: "missing tag checkout without refs",
			opts: &CloneOptions{Checkout: "refs/tags/missing"},
			assert: func(t *testing.T, _ *repo, err error) {
				require.ErrorContains(t, err, "invalid reference: refs/tags/missing")
			},
		},
		{
			name: "tag checkout without refs clones all branches and tags",
			opts: &CloneOptions{Checkout: "refs/tags/v1.0.0"},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				requireRefs(t, r.baseRepo, remoteBranchRefPrefix, "a", "b", "main")
				requireRefs(t, r.baseRepo, tagRefPrefix, "b", "v1.0.0")
				requireCurrentBranch(t, r, "")
				require.Equal(t, "main 1", subject(t, r.baseRepo, headRef))
			},
		},
		{
			name: "branch checkout does not fall back to a tag of the same name",
			opts: &CloneOptions{Checkout: "refs/heads/v1.0.0"},
			assert: func(t *testing.T, _ *repo, err error) {
				require.ErrorContains(t, err, `branch "v1.0.0" does not exist`)
			},
		},
		{
			name: "tag checkout is not mistaken for a branch of the same name",
			opts: &CloneOptions{
				Refs:     []string{"refs/tags/b"},
				Checkout: "refs/tags/b",
			},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				requireRefs(t, r.baseRepo, remoteBranchRefPrefix)
				requireRefs(t, r.baseRepo, tagRefPrefix, "b")
				requireCurrentBranch(t, r, "")
				require.Equal(t, "main 2", subject(t, r.baseRepo, headRef))
			},
		},
		{
			name: "commit checkout without refs clones all branches and tags",
			opts: &CloneOptions{Checkout: branchTip},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				requireRefs(t, r.baseRepo, remoteBranchRefPrefix, "a", "b", "main")
				requireRefs(t, r.baseRepo, tagRefPrefix, "b", "v1.0.0")
				requireCurrentBranch(t, r, "")
				require.Equal(t, "a", subject(t, r.baseRepo, headRef))
			},
		},
		{
			name: "commit checkout is cloned through the branch at that commit",
			opts: &CloneOptions{
				Refs:     []string{branchTip},
				Checkout: branchTip,
			},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				requireRefs(t, r.baseRepo, remoteBranchRefPrefix, "a")
				requireRefs(t, r.baseRepo, tagRefPrefix)
				requireCurrentBranch(t, r, "")
				require.Equal(t, "a", subject(t, r.baseRepo, headRef))
			},
		},
		{
			name: "commit checkout is cloned through the annotated tag at that commit",
			opts: &CloneOptions{
				Refs:     []string{taggedCommit},
				Checkout: taggedCommit,
			},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				requireRefs(t, r.baseRepo, remoteBranchRefPrefix)
				requireRefs(t, r.baseRepo, tagRefPrefix, "v1.0.0")
				requireCurrentBranch(t, r, "")
				require.Equal(t, "main 1", subject(t, r.baseRepo, headRef))
			},
		},
		{
			name: "commit the server will not serve by ID is found by searching",
			opts: &CloneOptions{
				Refs:     []string{unservedCommit},
				Checkout: unservedCommit,
			},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				requireRefs(t, r.baseRepo, remoteBranchRefPrefix, "a", "b", "main")
				requireRefs(t, r.baseRepo, tagRefPrefix, "b", "v1.0.0")
				requireCurrentBranch(t, r, "")
				require.Equal(t, "a 0", subject(t, r.baseRepo, headRef))
			},
		},
		{
			name: "depth without refs clones all branches and tags",
			opts: &CloneOptions{Depth: 1},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				requireRefs(t, r.baseRepo, remoteBranchRefPrefix, "a", "b", "main")
				requireRefs(t, r.baseRepo, tagRefPrefix, "b", "v1.0.0")
				require.True(t, isShallowRepo(t, r.baseRepo))
				require.Equal(t, 1, commitCount(t, r.baseRepo, "refs/remotes/origin/b"))
			},
		},
		{
			name: "depth applies to every cloned ref",
			opts: &CloneOptions{
				Refs:  []string{"refs/heads/a", "refs/tags/v1.0.0"},
				Depth: 1,
			},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				requireRefs(t, r.baseRepo, remoteBranchRefPrefix, "a", "main")
				requireRefs(t, r.baseRepo, tagRefPrefix, "v1.0.0")
				require.Equal(t, 1, commitCount(t, r.baseRepo, headRef))
				require.Equal(t, 1, commitCount(t, r.baseRepo, "refs/remotes/origin/a"))
				require.Equal(t, 1, commitCount(t, r.baseRepo, "refs/tags/v1.0.0"))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			rep, err := Clone(t.Context(), testRepoURL, clientOpts, testCase.opts)
			var r *repo
			if err == nil {
				t.Cleanup(func() { require.NoError(t, rep.Close(context.Background())) })
				var ok bool
				r, ok = rep.(*repo)
				require.True(t, ok)
			}
			testCase.assert(t, r, err)
		})
	}
}

func TestCloneBare(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(
		t,
		initMultipleBranches,
	)
	t.Cleanup(testServer.Close)
	clientOpts := &ClientOptions{Credentials: &testRepoCreds}
	branchTip, unservedCommit, taggedCommit := testCommitIDs(
		t,
		testRepoURL,
		clientOpts,
	)

	testCases := []struct {
		name   string
		opts   *BareCloneOptions
		assert func(*testing.T, *bareRepo, error)
	}{
		{
			name: "unsupported ref",
			opts: &BareCloneOptions{Refs: []string{"main"}},
			assert: func(t *testing.T, _ *bareRepo, err error) {
				require.ErrorContains(t, err, `unsupported ref "main"`)
			},
		},
		{
			name: "no refs clones all branches and tags",
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "a", "b", "main")
				requireRefs(t, b.baseRepo, tagRefPrefix, "b", "v1.0.0")
				require.False(t, isShallowRepo(t, b.baseRepo))
			},
		},
		{
			name: "HEAD clones only the default branch",
			opts: &BareCloneOptions{Refs: []string{headRef}},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "main")
				requireRefs(t, b.baseRepo, tagRefPrefix)
			},
		},
		{
			name: "only the specified branches are cloned",
			opts: &BareCloneOptions{
				Refs: []string{"refs/heads/b", "refs/heads/a"},
			},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "a", "b")
				requireRefs(t, b.baseRepo, tagRefPrefix)
			},
		},
		{
			name: "HEAD and a branch",
			opts: &BareCloneOptions{
				Refs: []string{headRef, "refs/heads/a"},
			},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "a", "main")
			},
		},
		{
			name: "refs missing from the remote are skipped",
			opts: &BareCloneOptions{
				Refs: []string{"refs/heads/missing", "refs/heads/a"},
			},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "a")
			},
		},
		{
			name: "default branch is cloned if no refs exist",
			opts: &BareCloneOptions{
				Refs: []string{"refs/heads/missing", "refs/tags/missing"},
			},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "main")
				requireRefs(t, b.baseRepo, tagRefPrefix)
			},
		},
		{
			name: "tags and branches",
			opts: &BareCloneOptions{
				Refs: []string{"refs/tags/v1.0.0", "refs/heads/a"},
			},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "a")
				requireRefs(t, b.baseRepo, tagRefPrefix, "v1.0.0")
			},
		},
		{
			name: "branch work tree can be added when HEAD is detached",
			opts: &BareCloneOptions{
				Refs: []string{"refs/tags/v1.0.0", "refs/heads/a"},
			},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				// Cloning the tag first leaves HEAD detached
				require.Equal(
					t,
					headRef,
					gitOutput(t, b.baseRepo, "rev-parse", "--symbolic-full-name", headRef),
				)
				workTree, err := b.AddWorkTree(
					t.Context(),
					filepath.Join(b.HomeDir(), "work-tree"),
					&AddWorkTreeOptions{Ref: "a"},
				)
				require.NoError(t, err)
				branch, err := workTree.CurrentBranch(t.Context())
				require.NoError(t, err)
				require.Equal(t, "a", branch)
				commitID, err := workTree.LastCommitID(t.Context())
				require.NoError(t, err)
				require.Equal(t, revParse(t, b.baseRepo, "refs/heads/a"), commitID)
			},
		},
		{
			name: "tag is not mistaken for a branch of the same name",
			opts: &BareCloneOptions{Refs: []string{"refs/tags/b"}},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix)
				requireRefs(t, b.baseRepo, tagRefPrefix, "b")
				require.Equal(t, "main 2", subject(t, b.baseRepo, "refs/tags/b"))
			},
		},
		{
			name: "commits are cloned through the branch at that commit",
			opts: &BareCloneOptions{
				Refs: []string{branchTip, "refs/heads/b"},
			},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "a", "b")
				requireRefs(t, b.baseRepo, tagRefPrefix)
				// HEAD is detached at the first ref, which is the commit
				require.Equal(
					t,
					headRef,
					gitOutput(t, b.baseRepo, "rev-parse", "--symbolic-full-name", headRef),
				)
				require.Equal(t, branchTip, revParse(t, b.baseRepo, headRef))
			},
		},
		{
			name: "commits are cloned through the annotated tag at that commit",
			opts: &BareCloneOptions{
				Refs: []string{"refs/heads/b", taggedCommit},
			},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "b")
				requireRefs(t, b.baseRepo, tagRefPrefix, "v1.0.0")
				require.Equal(t, "main 1", subject(t, b.baseRepo, taggedCommit))
			},
		},
		{
			name: "commit the server will not serve by ID is found by searching",
			opts: &BareCloneOptions{
				Refs: []string{headRef, unservedCommit},
			},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "a", "b", "main")
				requireRefs(t, b.baseRepo, tagRefPrefix, "b", "v1.0.0")
				require.Equal(t, "a 0", subject(t, b.baseRepo, unservedCommit))
				// HEAD comes first and exists, so HEAD is not detached
				require.Equal(
					t,
					"refs/heads/main",
					gitOutput(t, b.baseRepo, "rev-parse", "--symbolic-full-name", headRef),
				)
			},
		},
		{
			name: "HEAD is detached at a commit that follows only missing refs",
			opts: &BareCloneOptions{
				Refs: []string{"refs/heads/missing", unservedCommit},
			},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					headRef,
					gitOutput(t, b.baseRepo, "rev-parse", "--symbolic-full-name", headRef),
				)
				require.Equal(t, unservedCommit, revParse(t, b.baseRepo, headRef))
			},
		},
		{
			name: "HEAD is detached at a branch tip that follows only missing refs",
			opts: &BareCloneOptions{
				Refs: []string{"refs/heads/missing", branchTip},
			},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "a")
				require.Equal(
					t,
					headRef,
					gitOutput(t, b.baseRepo, "rev-parse", "--symbolic-full-name", headRef),
				)
				require.Equal(t, branchTip, revParse(t, b.baseRepo, headRef))
			},
		},
		{
			name: "depth without refs clones all branches and tags",
			opts: &BareCloneOptions{Depth: 1},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "a", "b", "main")
				requireRefs(t, b.baseRepo, tagRefPrefix, "b", "v1.0.0")
				require.True(t, isShallowRepo(t, b.baseRepo))
				require.Equal(t, 1, commitCount(t, b.baseRepo, "refs/heads/b"))
			},
		},
		{
			name: "depth applies to every cloned ref",
			opts: &BareCloneOptions{
				Refs:  []string{"refs/heads/a", "refs/heads/b", "refs/tags/v1.0.0"},
				Depth: 1,
			},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "a", "b")
				require.Equal(t, 1, commitCount(t, b.baseRepo, "refs/heads/a"))
				require.Equal(t, 1, commitCount(t, b.baseRepo, "refs/heads/b"))
				require.Equal(t, 1, commitCount(t, b.baseRepo, "refs/tags/v1.0.0"))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			rep, err := CloneBare(t.Context(), testRepoURL, clientOpts, testCase.opts)
			var b *bareRepo
			if err == nil {
				t.Cleanup(func() { require.NoError(t, rep.Close(context.Background())) })
				var ok bool
				b, ok = rep.(*bareRepo)
				require.True(t, ok)
			}
			testCase.assert(t, b, err)
		})
	}
}

func TestClone_commitReachability(t *testing.T) {
	var forkCommit, tagOnlyCommit, deepCommit string
	testServer, testRepoURL, testRepoCreds := setupRemoteRepoWithConfig(
		t,
		map[string]string{
			// Like GitHub, serve any commit by ID, whether or not a branch or tag
			// of the repository reaches it
			"uploadpack.allowAnySHA1InWant": "true",
			// Like GitHub, support blobless clones, which fetch any missing object
			// by its ID when it is needed
			"uploadpack.allowFilter": "true",
		},
		func(t *testing.T, rep WorkTree) {
			forkCommit, tagOnlyCommit, deepCommit = initUnreachableCommits(t, rep)
		},
	)
	t.Cleanup(testServer.Close)
	clientOpts := &ClientOptions{Credentials: &testRepoCreds}

	clone := func(opts *CloneOptions) func(*testing.T) (*baseRepo, error) {
		return func(t *testing.T) (*baseRepo, error) {
			rep, err := Clone(t.Context(), testRepoURL, clientOpts, opts)
			if err != nil {
				return nil, err
			}
			t.Cleanup(func() { require.NoError(t, rep.Close(context.Background())) })
			r, ok := rep.(*repo)
			require.True(t, ok)
			return r.baseRepo, nil
		}
	}
	cloneBare := func(opts *BareCloneOptions) func(*testing.T) (*baseRepo, error) {
		return func(t *testing.T) (*baseRepo, error) {
			rep, err := CloneBare(t.Context(), testRepoURL, clientOpts, opts)
			if err != nil {
				return nil, err
			}
			t.Cleanup(func() { require.NoError(t, rep.Close(context.Background())) })
			b, ok := rep.(*bareRepo)
			require.True(t, ok)
			return b.baseRepo, nil
		}
	}

	// The server serves the fork's commit when asked for it by ID, so the cases
	// below that fail to clone it fail only because Kargo never asks
	fetched, err := clone(nil)(t)
	require.NoError(t, err)
	gitOutput(t, fetched, "fetch", "origin", forkCommit)
	require.Equal(t, "fork", subject(t, fetched, forkCommit))

	requireUnreachable := func(t *testing.T, _ *baseRepo, err error) {
		require.ErrorContains(t, err, "no branch or tag in remote repo")
		require.ErrorContains(t, err, forkCommit)
	}
	testCases := []struct {
		name   string
		clone  func(*testing.T) (*baseRepo, error)
		assert func(*testing.T, *baseRepo, error)
	}{
		{
			name:   "fork's commit checked out without refs",
			clone:  clone(&CloneOptions{Checkout: forkCommit}),
			assert: requireUnreachable,
		},
		{
			name: "fork's commit checked out at depth 1",
			clone: clone(&CloneOptions{
				Refs:     []string{forkCommit},
				Checkout: forkCommit,
				Depth:    1,
			}),
			assert: requireUnreachable,
		},
		{
			name: "fork's commit in a bare clone",
			clone: cloneBare(&BareCloneOptions{
				Refs: []string{headRef, forkCommit},
			}),
			assert: requireUnreachable,
		},
		{
			name: "fork's commit in a blobless clone",
			clone: clone(&CloneOptions{
				Refs:     []string{forkCommit},
				Checkout: forkCommit,
				Blobless: true,
			}),
			assert: requireUnreachable,
		},
		{
			name: "commit only a tag reaches",
			clone: cloneBare(&BareCloneOptions{
				Refs:  []string{tagOnlyCommit},
				Depth: 1,
			}),
			assert: func(t *testing.T, b *baseRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b, tagRefPrefix, "t")
				require.Equal(t, tagOnlyCommit, revParse(t, b, headRef))
			},
		},
		{
			name: "commit only a tag reaches, with that tag cloned at depth 1",
			clone: clone(&CloneOptions{
				Refs:     []string{"refs/tags/t"},
				Checkout: tagOnlyCommit,
				Depth:    1,
			}),
			assert: func(t *testing.T, b *baseRepo, err error) {
				require.NoError(t, err)
				require.Equal(t, "tagged 0", subject(t, b, headRef))
			},
		},
		{
			name: "commit only a tag reaches, at depth 1 without refs",
			clone: clone(&CloneOptions{
				Checkout: tagOnlyCommit,
				Depth:    1,
			}),
			assert: func(t *testing.T, b *baseRepo, err error) {
				require.NoError(t, err)
				require.Equal(t, "tagged 0", subject(t, b, headRef))
			},
		},
		{
			name: "deepening stops when it brings in nothing new",
			clone: cloneBare(&BareCloneOptions{
				Refs:  []string{"refs/heads/main", "refs/tags/t"},
				Depth: 1,
			}),
			assert: func(t *testing.T, b *baseRepo, err error) {
				require.NoError(t, err)
				// The commit is behind the tag's shallow boundary, which deepening
				// along main never moves
				missing, err := b.deepenUntilPresent(
					t.Context(),
					[]string{tagOnlyCommit},
					refspecs([]string{"refs/heads/main"}, true),
				)
				require.NoError(t, err)
				require.Equal(t, []string{tagOnlyCommit}, missing)
				require.Equal(t, 40, commitCount(t, b, "refs/heads/main"))
				require.True(t, isShallowRepo(t, b))
			},
		},
		{
			name: "commit deep in a branch's history at depth 1",
			clone: clone(&CloneOptions{
				Refs:     []string{deepCommit},
				Checkout: deepCommit,
				Depth:    1,
			}),
			assert: func(t *testing.T, b *baseRepo, err error) {
				require.NoError(t, err)
				require.Equal(t, "main 25", subject(t, b, headRef))
				// Deepening stopped once the commit was found
				require.True(t, isShallowRepo(t, b))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			b, err := testCase.clone(t)
			testCase.assert(t, b, err)
		})
	}
}

func Test_validateCloneRefs(t *testing.T) {
	testCases := []struct {
		name   string
		refs   []string
		assert func(*testing.T, error)
	}{
		{
			name: "no refs",
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "every supported form",
			refs: []string{
				headRef,
				"refs/heads/main",
				"refs/heads/stage/dev",
				"refs/tags/v1.0.0",
				strings.Repeat("a", 40),
				strings.Repeat("A", 64),
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "short branch name",
			refs: []string{"main"},
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `unsupported ref "main"`)
			},
		},
		{
			name: "branch prefix without a name",
			refs: []string{"refs/heads/"},
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `unsupported ref "refs/heads/"`)
			},
		},
		{
			name: "tag prefix without a name",
			refs: []string{"refs/tags/"},
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `unsupported ref "refs/tags/"`)
			},
		},
		{
			name: "remote-tracking branch",
			refs: []string{"refs/remotes/origin/main"},
			assert: func(t *testing.T, err error) {
				require.ErrorContains(
					t, err, `unsupported ref "refs/remotes/origin/main"`,
				)
			},
		},
		{
			name: "abbreviated commit ID",
			refs: []string{"abc1234"},
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `unsupported ref "abc1234"`)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assert(t, validateCloneRefs(testCase.refs))
		})
	}
}

func Test_existingRefs(t *testing.T) {
	commitID := strings.Repeat("a", 40)
	otherCommitID := strings.Repeat("b", 40)
	testCases := []struct {
		name          string
		refs          []string
		remoteRefs    map[string]string
		defaultBranch string
		assert        func(*testing.T, []string, error)
	}{
		{
			name: "existing refs in order, missing refs skipped",
			refs: []string{
				"refs/tags/v1.0.0",
				"refs/heads/missing",
				"refs/heads/a",
			},
			remoteRefs: map[string]string{
				"refs/heads/a":     commitID,
				"refs/tags/v1.0.0": otherCommitID,
			},
			assert: func(t *testing.T, existing []string, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					[]string{"refs/tags/v1.0.0", "refs/heads/a"},
					existing,
				)
			},
		},
		{
			name: "commit replaced by the first ref by name at that commit",
			refs: []string{commitID},
			remoteRefs: map[string]string{
				"refs/tags/v1.0.0": commitID,
				"refs/heads/b":     commitID,
				"refs/heads/a":     commitID,
				"refs/heads/main":  otherCommitID,
			},
			assert: func(t *testing.T, existing []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"refs/heads/a"}, existing)
			},
		},
		{
			name: "commit replaced by a requested ref at that commit, once",
			refs: []string{commitID, "refs/tags/v1.0.0"},
			remoteRefs: map[string]string{
				"refs/heads/a":     commitID,
				"refs/tags/v1.0.0": commitID,
			},
			assert: func(t *testing.T, existing []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"refs/tags/v1.0.0"}, existing)
			},
		},
		{
			name: "commit at no branch or tag skipped",
			refs: []string{commitID, "refs/heads/a"},
			remoteRefs: map[string]string{
				"refs/heads/a": otherCommitID,
				// Only branches and tags can stand in for a commit
				"refs/pull/1/head": commitID,
			},
			assert: func(t *testing.T, existing []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"refs/heads/a"}, existing)
			},
		},
		{
			name:          "HEAD resolved to the default branch, once",
			refs:          []string{headRef, "refs/heads/main"},
			remoteRefs:    map[string]string{"refs/heads/main": commitID},
			defaultBranch: "refs/heads/main",
			assert: func(t *testing.T, existing []string, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"refs/heads/main"}, existing)
			},
		},
		{
			name: "HEAD in an empty remote",
			refs: []string{headRef},
			assert: func(t *testing.T, existing []string, err error) {
				require.NoError(t, err)
				require.Empty(t, existing)
			},
		},
		{
			name:       "HEAD with no default branch reported",
			refs:       []string{"refs/heads/a", headRef},
			remoteRefs: map[string]string{"refs/heads/a": commitID},
			assert: func(t *testing.T, _ []string, err error) {
				require.ErrorContains(t, err, "cannot determine the default branch")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			existing, err := existingRefs(
				testCase.refs,
				testCase.remoteRefs,
				testCase.defaultBranch,
			)
			testCase.assert(t, existing, err)
		})
	}
}

func Test_firstCommit(t *testing.T) {
	commitID := strings.Repeat("a", 40)
	testCases := []struct {
		name          string
		refs          []string
		existing      []string
		defaultBranch string
		expected      string
	}{
		{
			name:     "commit first",
			refs:     []string{commitID, "refs/heads/a"},
			existing: []string{"refs/heads/a"},
			expected: commitID,
		},
		{
			name:     "commit after missing refs",
			refs:     []string{"refs/heads/missing", "refs/tags/missing", commitID},
			expected: commitID,
		},
		{
			name:     "commit after an existing ref",
			refs:     []string{"refs/heads/a", commitID},
			existing: []string{"refs/heads/a"},
		},
		{
			name:          "commit after HEAD",
			refs:          []string{headRef, commitID},
			existing:      []string{"refs/heads/main"},
			defaultBranch: "refs/heads/main",
		},
		{
			name:     "commit after HEAD in an empty remote",
			refs:     []string{headRef, commitID},
			expected: commitID,
		},
		{
			name:     "no commit",
			refs:     []string{"refs/heads/missing"},
			existing: []string{},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.Equal(
				t,
				testCase.expected,
				firstCommit(
					testCase.refs,
					testCase.existing,
					testCase.defaultBranch,
				),
			)
		})
	}
}

func Test_uniqueRefs(t *testing.T) {
	require.Equal(
		t,
		[]string{"refs/heads/b", headRef, "refs/heads/a"},
		uniqueRefs([]string{
			"refs/heads/b",
			headRef,
			"refs/heads/b",
			"refs/heads/a",
			headRef,
		}),
	)
}

func Test_parseLsRemoteSymrefOutput(t *testing.T) {
	testCases := []struct {
		name   string
		output string
		assert func(*testing.T, map[string]string, string, error)
	}{
		{
			name: "empty remote",
			assert: func(t *testing.T, refs map[string]string, head string, err error) {
				require.NoError(t, err)
				require.Empty(t, refs)
				require.Empty(t, head)
			},
		},
		{
			name: "HEAD, branches, and tags",
			output: "ref: refs/heads/main\tHEAD\n" +
				"1111111111111111111111111111111111111111\tHEAD\n" +
				"1111111111111111111111111111111111111111\trefs/heads/main\n" +
				"2222222222222222222222222222222222222222\trefs/heads/a\n" +
				"3333333333333333333333333333333333333333\trefs/tags/v1.0.0\n" +
				"4444444444444444444444444444444444444444\trefs/tags/v1.0.0^{}\n" +
				"5555555555555555555555555555555555555555\trefs/tags/light\n",
			assert: func(t *testing.T, refs map[string]string, head string, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					map[string]string{
						headRef:            "1111111111111111111111111111111111111111",
						"refs/heads/main":  "1111111111111111111111111111111111111111",
						"refs/heads/a":     "2222222222222222222222222222222222222222",
						"refs/tags/v1.0.0": "4444444444444444444444444444444444444444",
						"refs/tags/light":  "5555555555555555555555555555555555555555",
					},
					refs,
				)
				require.Equal(t, "refs/heads/main", head)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			refs, head, err := parseLsRemoteSymrefOutput([]byte(testCase.output))
			testCase.assert(t, refs, head, err)
		})
	}
}

// initMultipleBranches pushes commits "main 0" through "main 2" to main, with
// tag v1.0.0 at "main 1" and tag b at "main 2". It then pushes branch a, with
// commits "a 0" and "a" ahead of main, and branch b, with commit "b" ahead of
// main.
func initMultipleBranches(t *testing.T, rep WorkTree) {
	commit := func(msg string) {
		require.NoError(t, os.WriteFile(
			filepath.Join(rep.Dir(), "test.txt"),
			[]byte(msg),
			0o600,
		))
		require.NoError(t, rep.AddAllAndCommit(t.Context(), msg, nil))
	}
	tag := func(name string) {
		require.NoError(t, rep.CreateTag(t.Context(), name, name, nil))
		require.NoError(t, rep.Push(t.Context(), &PushOptions{Tag: name}))
	}
	for i := range 3 {
		commit(fmt.Sprintf("main %d", i))
		if i == 1 {
			tag("v1.0.0")
		}
	}
	require.NoError(t, rep.Push(t.Context(), nil))
	for _, branch := range []string{"a", "b"} {
		require.NoError(t, rep.CreateChildBranch(t.Context(), branch))
		if branch == "a" {
			commit("a 0")
		}
		commit(branch)
		require.NoError(t, rep.Push(t.Context(), nil))
		require.NoError(t, rep.Checkout(t.Context(), "main"))
	}
	// Pushing to branch b would be ambiguous once tag b exists
	tag("b")
}

// initUnreachableCommits pushes commits "main 0" through "main 39" to main. It
// then pushes commit "fork" to branch fork and deletes the branch, so that the
// server still has the commit but no branch or tag reaches it, as with a
// commit that exists only in a fork. Last, it pushes tag t at commit
// "tagged 1", whose parent, "tagged 0", only the tag reaches. It returns the
// IDs of "fork", "tagged 0", and "main 25".
func initUnreachableCommits(
	t *testing.T,
	rep WorkTree,
) (string, string, string) {
	commit := func(msg string) string {
		require.NoError(t, os.WriteFile(
			filepath.Join(rep.Dir(), "test.txt"),
			[]byte(msg),
			0o600,
		))
		require.NoError(t, rep.AddAllAndCommit(t.Context(), msg, nil))
		id, err := rep.LastCommitID(t.Context())
		require.NoError(t, err)
		return id
	}
	var deepCommit string
	for i := range 40 {
		if id := commit(fmt.Sprintf("main %d", i)); i == 25 {
			deepCommit = id
		}
	}
	require.NoError(t, rep.Push(t.Context(), nil))

	require.NoError(t, rep.CreateChildBranch(t.Context(), "fork"))
	forkCommit := commit("fork")
	require.NoError(t, rep.Push(t.Context(), nil))
	r, ok := rep.(*repo)
	require.True(t, ok)
	gitOutput(t, r.baseRepo, "push", "--delete", "origin", "fork")

	require.NoError(t, rep.Checkout(t.Context(), "main"))
	require.NoError(t, rep.CreateChildBranch(t.Context(), "tagged"))
	tagOnlyCommit := commit("tagged 0")
	commit("tagged 1")
	require.NoError(t, rep.CreateTag(t.Context(), "t", "t", nil))
	require.NoError(t, rep.Push(t.Context(), &PushOptions{Tag: "t"}))
	require.NoError(t, rep.Checkout(t.Context(), "main"))
	return forkCommit, tagOnlyCommit, deepCommit
}

// testCommitIDs returns the ID of the commit at the tip of branch a, the ID of
// commit "a 0", which no ref points to and which is not in main's history, and
// the ID of commit "main 1", which annotated tag v1.0.0 points to. A server
// that does not allow fetching arbitrary commits by ID will refuse to serve
// "a 0" by its ID.
func testCommitIDs(
	t *testing.T,
	repoURL string,
	clientOpts *ClientOptions,
) (string, string, string) {
	t.Helper()
	rep, err := Clone(t.Context(), repoURL, clientOpts, nil)
	require.NoError(t, err)
	defer func() { require.NoError(t, rep.Close(t.Context())) }()
	r, ok := rep.(*repo)
	require.True(t, ok)
	return revParse(t, r.baseRepo, "refs/remotes/origin/a"),
		revParse(t, r.baseRepo, "refs/remotes/origin/a~1"),
		revParse(t, r.baseRepo, "refs/tags/v1.0.0^{commit}")
}

// requireRefs asserts that the names of the refs stored under prefix in the
// repository are exactly the specified ones.
func requireRefs(
	t *testing.T,
	b *baseRepo,
	prefix string,
	names ...string,
) {
	t.Helper()
	out, err := libExec.Exec(b.buildGitCommand(
		t.Context(),
		"for-each-ref", "--format=%(refname)", prefix,
	))
	require.NoError(t, err)
	var actual []string
	for ref := range strings.Lines(string(out)) {
		name := strings.TrimPrefix(strings.TrimSpace(ref), prefix)
		if name != headRef { // Ignore refs/remotes/origin/HEAD
			actual = append(actual, name)
		}
	}
	require.ElementsMatch(t, names, actual)
}

func requireCurrentBranch(t *testing.T, r *repo, branch string) {
	t.Helper()
	current, err := r.CurrentBranch(t.Context())
	require.NoError(t, err)
	require.Equal(t, branch, current)
}

func isShallowRepo(t *testing.T, b *baseRepo) bool {
	t.Helper()
	return gitOutput(t, b, "rev-parse", "--is-shallow-repository") == "true"
}

func commitCount(t *testing.T, b *baseRepo, rev string) int {
	t.Helper()
	count, err := strconv.Atoi(gitOutput(t, b, "rev-list", "--count", rev))
	require.NoError(t, err)
	return count
}

func revParse(t *testing.T, b *baseRepo, rev string) string {
	t.Helper()
	return gitOutput(t, b, "rev-parse", rev)
}

// subject returns the subject of the commit that rev resolves to.
func subject(t *testing.T, b *baseRepo, rev string) string {
	t.Helper()
	return gitOutput(t, b, "log", "-1", "--format=%s", rev)
}

func gitOutput(t *testing.T, b *baseRepo, args ...string) string {
	t.Helper()
	out, err := libExec.Exec(b.buildGitCommand(t.Context(), args...))
	require.NoError(t, err)
	return strings.TrimSpace(string(out))
}
