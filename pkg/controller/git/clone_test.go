package git

import (
	"context"
	"errors"
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
	branchTip, unservedCommit := testCommitIDs(t, testRepoURL, clientOpts)

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
			name: "commit checkout",
			opts: &CloneOptions{
				Refs:     []string{branchTip},
				Checkout: branchTip,
			},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				requireRefs(t, r.baseRepo, remoteBranchRefPrefix)
				requireRefs(t, r.baseRepo, tagRefPrefix)
				require.Equal(t, "a", subject(t, r.baseRepo, headRef))
			},
		},
		{
			name: "commit the server will not serve by ID",
			opts: &CloneOptions{
				Refs:     []string{unservedCommit},
				Checkout: unservedCommit,
			},
			assert: func(t *testing.T, _ *repo, err error) {
				require.ErrorContains(
					t, err, "server does not support fetching commits by ID",
				)
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
	branchTip, unservedCommit := testCommitIDs(t, testRepoURL, clientOpts)

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
			name: "commits and branches",
			opts: &BareCloneOptions{
				Refs: []string{branchTip, "refs/heads/b"},
			},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "b")
				requireRefs(t, b.baseRepo, tagRefPrefix)
				require.Equal(t, "a", subject(t, b.baseRepo, branchTip))
			},
		},
		{
			name: "commit the server will not serve by ID",
			opts: &BareCloneOptions{
				Refs: []string{headRef, unservedCommit},
			},
			assert: func(t *testing.T, _ *bareRepo, err error) {
				require.ErrorContains(
					t, err, "server does not support fetching commits by ID",
				)
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

func Test_explainCommitFetchError(t *testing.T) {
	testCases := []struct {
		name   string
		err    error
		assert func(*testing.T, error)
	}{
		{
			name: "server refused a commit requested by ID",
			err: &libExec.ExitError{
				Output: []byte(
					"error: Server does not allow request for unadvertised object abc",
				),
				ExitCode: 128,
			},
			assert: func(t *testing.T, err error) {
				require.ErrorContains(
					t, err, "server does not support fetching commits by ID",
				)
				var exitErr *libExec.ExitError
				require.ErrorAs(t, err, &exitErr)
			},
		},
		{
			name: "any other error",
			err:  errors.New("something went wrong"),
			assert: func(t *testing.T, err error) {
				require.EqualError(t, err, "something went wrong")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assert(t, explainCommitFetchError(testCase.err))
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
		assert func(*testing.T, map[string]struct{}, string, error)
	}{
		{
			name: "empty remote",
			assert: func(t *testing.T, refs map[string]struct{}, head string, err error) {
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
				"3333333333333333333333333333333333333333\trefs/tags/v1.0.0\n",
			assert: func(t *testing.T, refs map[string]struct{}, head string, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					map[string]struct{}{
						headRef:            {},
						"refs/heads/main":  {},
						"refs/heads/a":     {},
						"refs/tags/v1.0.0": {},
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

// testCommitIDs returns the ID of the commit at the tip of branch a, which a
// server will serve by ID because a branch points to it, and the ID of commit
// "a 0", which no ref points to and which is not in main's history. A server
// that does not allow fetching arbitrary commits by ID will refuse to serve
// it.
func testCommitIDs(
	t *testing.T,
	repoURL string,
	clientOpts *ClientOptions,
) (string, string) {
	t.Helper()
	rep, err := Clone(t.Context(), repoURL, clientOpts, nil)
	require.NoError(t, err)
	defer func() { require.NoError(t, rep.Close(t.Context())) }()
	r, ok := rep.(*repo)
	require.True(t, ok)
	return revParse(t, r.baseRepo, "refs/remotes/origin/a"),
		revParse(t, r.baseRepo, "refs/remotes/origin/a~1")
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
