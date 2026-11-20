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
			name: "no refs clones all branches and tags",
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				requireRefs(t, r.baseRepo, remoteBranchRefPrefix, "a", "b", "main")
				requireRefs(t, r.baseRepo, tagRefPrefix, "v1.0.0")
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
			name: "branch to check out is cloned whether or not refs include it",
			opts: &CloneOptions{
				Refs:   []string{"refs/heads/a"},
				Branch: "b",
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
			name: "depth without refs clones all branches",
			opts: &CloneOptions{Depth: 1},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				requireRefs(t, r.baseRepo, remoteBranchRefPrefix, "a", "b", "main")
				requireRefs(t, r.baseRepo, tagRefPrefix, "v1.0.0")
				require.True(t, isShallowRepo(t, r.baseRepo))
				require.Equal(t, 1, commitCount(t, r.baseRepo, "refs/remotes/origin/b"))
			},
		},
		{
			name: "depth applies to every cloned ref",
			opts: &CloneOptions{
				Refs:  []string{"refs/heads/a"},
				Depth: 1,
			},
			assert: func(t *testing.T, r *repo, err error) {
				require.NoError(t, err)
				requireRefs(t, r.baseRepo, remoteBranchRefPrefix, "a", "main")
				require.Equal(t, 1, commitCount(t, r.baseRepo, "HEAD"))
				require.Equal(t, 1, commitCount(t, r.baseRepo, "refs/remotes/origin/a"))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			rep, err := Clone(
				t.Context(),
				testRepoURL,
				&ClientOptions{Credentials: &testRepoCreds},
				testCase.opts,
			)
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

	testCases := []struct {
		name   string
		opts   *BareCloneOptions
		assert func(*testing.T, *bareRepo, error)
	}{
		{
			name: "unsupported ref",
			opts: &BareCloneOptions{Refs: []string{"refs/tags/v1.0.0"}},
			assert: func(t *testing.T, _ *bareRepo, err error) {
				require.ErrorContains(t, err, `unsupported ref "refs/tags/v1.0.0"`)
			},
		},
		{
			name: "no refs clones all branches and tags",
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "a", "b", "main")
				requireRefs(t, b.baseRepo, tagRefPrefix, "v1.0.0")
				require.False(t, isShallowRepo(t, b.baseRepo))
			},
		},
		{
			name: "HEAD clones only the default branch",
			opts: &BareCloneOptions{Refs: []string{headRef}},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "main")
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
			opts: &BareCloneOptions{Refs: []string{"refs/heads/missing"}},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "main")
			},
		},
		{
			name: "depth without refs clones all branches",
			opts: &BareCloneOptions{Depth: 1},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "a", "b", "main")
				requireRefs(t, b.baseRepo, tagRefPrefix, "v1.0.0")
				require.True(t, isShallowRepo(t, b.baseRepo))
				require.Equal(t, 1, commitCount(t, b.baseRepo, "refs/heads/b"))
			},
		},
		{
			name: "depth applies to every cloned ref",
			opts: &BareCloneOptions{
				Refs:  []string{"refs/heads/a", "refs/heads/b"},
				Depth: 1,
			},
			assert: func(t *testing.T, b *bareRepo, err error) {
				require.NoError(t, err)
				requireRefs(t, b.baseRepo, branchRefPrefix, "a", "b")
				require.Equal(t, 1, commitCount(t, b.baseRepo, "refs/heads/a"))
				require.Equal(t, 1, commitCount(t, b.baseRepo, "refs/heads/b"))
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			rep, err := CloneBare(
				t.Context(),
				testRepoURL,
				&ClientOptions{Credentials: &testRepoCreds},
				testCase.opts,
			)
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
			name: "HEAD and branches",
			refs: []string{headRef, "refs/heads/main", "refs/heads/stage/dev"},
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
			name: "tag",
			refs: []string{"refs/tags/v1.0.0"},
			assert: func(t *testing.T, err error) {
				require.ErrorContains(t, err, `unsupported ref "refs/tags/v1.0.0"`)
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
			name: "HEAD and branches",
			output: "ref: refs/heads/main\tHEAD\n" +
				"1111111111111111111111111111111111111111\tHEAD\n" +
				"1111111111111111111111111111111111111111\trefs/heads/main\n" +
				"2222222222222222222222222222222222222222\trefs/heads/a\n",
			assert: func(t *testing.T, refs map[string]struct{}, head string, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					map[string]struct{}{
						headRef:           {},
						"refs/heads/main": {},
						"refs/heads/a":    {},
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

// initMultipleBranches pushes three commits to main, tagged v1.0.0, then two
// branches, a and b, each one commit ahead of main.
func initMultipleBranches(t *testing.T, rep WorkTree) {
	commit := func(msg string) {
		require.NoError(t, os.WriteFile(
			filepath.Join(rep.Dir(), "test.txt"),
			[]byte(msg),
			0o600,
		))
		require.NoError(t, rep.AddAllAndCommit(t.Context(), msg, nil))
	}
	for i := range 3 {
		commit(fmt.Sprintf("main %d", i))
	}
	require.NoError(t, rep.Push(t.Context(), nil))
	require.NoError(t, rep.CreateTag(t.Context(), "v1.0.0", "", nil))
	require.NoError(t, rep.Push(t.Context(), &PushOptions{Tag: "v1.0.0"}))
	for _, branch := range []string{"a", "b"} {
		require.NoError(t, rep.CreateChildBranch(t.Context(), branch))
		commit(branch)
		require.NoError(t, rep.Push(t.Context(), nil))
		require.NoError(t, rep.Checkout(t.Context(), "main"))
	}
}

const tagRefPrefix = "refs/tags/"

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
	out, err := libExec.Exec(b.buildGitCommand(
		t.Context(),
		"rev-parse", "--is-shallow-repository",
	))
	require.NoError(t, err)
	return strings.TrimSpace(string(out)) == "true"
}

func commitCount(t *testing.T, b *baseRepo, ref string) int {
	t.Helper()
	out, err := libExec.Exec(b.buildGitCommand(
		t.Context(),
		"rev-list", "--count", ref,
	))
	require.NoError(t, err)
	count, err := strconv.Atoi(strings.TrimSpace(string(out)))
	require.NoError(t, err)
	return count
}
