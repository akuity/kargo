package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(
		os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
	return strings.TrimSpace(string(out))
}

// TestShallowBareClonePromotionFlow runs the git-clone, git-commit and git-push
// promotion steps' git operations against a shallow clone, while other
// promotions push to the same branch between our clone and our push.
func TestShallowBareClonePromotionFlow(t *testing.T) {
	tmp := t.TempDir()

	// A remote with a long history on main plus an unrelated branch.
	remote := filepath.Join(tmp, "remote.git")
	runGit(t, tmp, "init", "--bare", "--initial-branch=main", remote)
	seed := filepath.Join(tmp, "seed")
	runGit(t, tmp, "clone", remote, seed)
	for i := 0; i < 50; i++ {
		f := filepath.Join(seed, "apps", fmt.Sprintf("app-%d", i%5), "values.yaml")
		require.NoError(t, os.MkdirAll(filepath.Dir(f), 0o755))
		require.NoError(t, os.WriteFile(f, []byte(fmt.Sprintf("tag: v%d\n", i)), 0o600))
		runGit(t, seed, "add", "-A")
		runGit(t, seed, "commit", "-m", fmt.Sprintf("promote %d", i))
	}
	runGit(t, seed, "push", "origin", "HEAD:main")
	runGit(t, seed, "push", "origin", "HEAD:other-branch")

	// file:// so that git honors --depth for a local remote.
	url := "file://" + remote
	repo, err := CloneBare(
		url,
		&ClientOptions{User: &User{Name: "kargo", Email: "kargo@example.com"}},
		&BareCloneOptions{BaseDir: tmp, Branch: "main", Depth: 1},
	)
	require.NoError(t, err)
	defer repo.Close()

	require.Equal(t, "true", runGit(t, repo.Dir(), "rev-parse", "--is-shallow-repository"))
	require.Equal(t, "1", runGit(t, repo.Dir(), "rev-list", "--count", "main"))
	require.Equal(t, "main", runGit(t, repo.Dir(), "for-each-ref", "--format=%(refname:short)", "refs/heads"))

	exists, err := repo.RemoteBranchExists("main")
	require.NoError(t, err)
	require.True(t, exists)

	wt, err := repo.AddWorkTree(filepath.Join(tmp, "out"), &AddWorkTreeOptions{Ref: "main"})
	require.NoError(t, err)

	// Other promotions push to main after our clone, touching other files.
	for i := 0; i < 3; i++ {
		f := filepath.Join(seed, "apps", "app-1", "values.yaml")
		require.NoError(t, os.WriteFile(f, []byte(fmt.Sprintf("tag: other-%d\n", i)), 0o600))
		runGit(t, seed, "commit", "-am", fmt.Sprintf("other promotion %d", i))
		runGit(t, seed, "push", "origin", "HEAD:main")
	}

	f := filepath.Join(wt.Dir(), "apps", "app-3", "values.yaml")
	require.NoError(t, os.WriteFile(f, []byte("tag: ours\n"), 0o600))
	require.NoError(t, wt.AddAllAndCommit("our promotion"))
	require.NoError(t, wt.Push(&PushOptions{TargetBranch: "main", PullRebase: true}))

	// Remote main has full history plus all four promotions, ours on top.
	require.Equal(t, "54", runGit(t, remote, "rev-list", "--count", "main"))
	require.Equal(t, "our promotion", runGit(t, remote, "log", "-1", "--format=%s", "main"))
	require.Equal(t, "tag: ours", runGit(t, remote, "show", "main:apps/app-3/values.yaml"))
	require.Equal(t, "tag: other-2", runGit(t, remote, "show", "main:apps/app-1/values.yaml"))
	runGit(t, remote, "fsck", "--strict")
}

func TestBareCloneWithoutBranchIsFull(t *testing.T) {
	tmp := t.TempDir()
	remote := filepath.Join(tmp, "remote.git")
	runGit(t, tmp, "init", "--bare", "--initial-branch=main", remote)
	seed := filepath.Join(tmp, "seed")
	runGit(t, tmp, "clone", remote, seed)
	for i := 0; i < 3; i++ {
		runGit(t, seed, "commit", "--allow-empty", "-m", fmt.Sprintf("c%d", i))
	}
	runGit(t, seed, "push", "origin", "HEAD:main")
	runGit(t, seed, "push", "origin", "HEAD:other-branch")

	repo, err := CloneBare("file://"+remote, nil, &BareCloneOptions{BaseDir: tmp, Depth: 1})
	require.NoError(t, err)
	defer repo.Close()
	require.Equal(t, "false", runGit(t, repo.Dir(), "rev-parse", "--is-shallow-repository"))
	require.Equal(t, "3", runGit(t, repo.Dir(), "rev-list", "--count", "main"))
	require.Equal(t, "main\nother-branch", runGit(t, repo.Dir(), "for-each-ref", "--format=%(refname:short)", "refs/heads"))
}
