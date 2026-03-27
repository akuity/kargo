package git

import (
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	libExec "github.com/akuity/kargo/pkg/exec"
)

func TestRepo(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(t)
	defer testServer.Close()

	rep, err := Clone(
		testRepoURL,
		&ClientOptions{
			Credentials: &testRepoCreds,
		},
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, rep)
	defer rep.Close()
	r, ok := rep.(*repo)
	require.True(t, ok)

	t.Run("can clone", func(t *testing.T) {
		require.Equal(t, testRepoURL, r.originalURL)
		var accessURL *url.URL
		accessURL, err = url.Parse(r.accessURL)
		require.NoError(t, err)
		accessURL.User = nil
		require.Equal(t, testRepoURL, accessURL.String())
		require.NotEmpty(t, r.homeDir)
		var fi os.FileInfo
		fi, err = os.Stat(r.homeDir)
		require.NoError(t, err)
		require.True(t, fi.IsDir())
		require.NotEmpty(t, r.dir)
		fi, err = os.Stat(r.dir)
		require.NoError(t, err)
		require.True(t, fi.IsDir())
	})

	t.Run("can get the repo url", func(t *testing.T) {
		require.Equal(t, r.originalURL, r.URL())
	})

	t.Run("can get the home dir", func(t *testing.T) {
		require.Equal(t, r.homeDir, r.HomeDir())
	})

	t.Run("can get the working tree dir", func(t *testing.T) {
		require.Equal(t, r.dir, r.Dir())
	})

	t.Run("can check for diffs -- negative result", func(t *testing.T) {
		var hasDiffs bool
		hasDiffs, err = rep.HasDiffs()
		require.NoError(t, err)
		require.False(t, hasDiffs)
	})

	err = os.WriteFile(fmt.Sprintf("%s/%s", rep.Dir(), "test.txt"), []byte("foo"), 0600)
	require.NoError(t, err)

	t.Run("can check for diffs -- positive result", func(t *testing.T) {
		var hasDiffs bool
		hasDiffs, err = rep.HasDiffs()
		require.NoError(t, err)
		require.True(t, hasDiffs)
	})

	testCommitMessage := fmt.Sprintf(`test commit %s

with a body
`, uuid.NewString())
	err = rep.AddAllAndCommit(testCommitMessage, nil)
	require.NoError(t, err)

	t.Run("can commit", func(t *testing.T) {
		require.NoError(t, err)
	})

	lastCommitID, err := rep.LastCommitID()
	require.NoError(t, err)

	t.Run("can get last commit id", func(t *testing.T) {
		require.NoError(t, err)
		require.NotEmpty(t, lastCommitID)
	})

	t.Run("can get commit message by id", func(t *testing.T) {
		var msg string
		msg, err = rep.CommitMessage(lastCommitID)
		require.NoError(t, err)
		require.Equal(t, testCommitMessage, msg)
	})

	t.Run("can get diff paths", func(t *testing.T) {
		var paths []string
		paths, err = rep.GetDiffPathsForCommitID(lastCommitID)
		require.NoError(t, err)
		require.Len(t, paths, 1)
	})

	t.Run("can check if remote branch exists -- negative result", func(t *testing.T) {
		var exists bool
		exists, err = rep.RemoteBranchExists("main") // The remote repo is empty!
		require.NoError(t, err)
		require.False(t, exists)
	})

	err = rep.Push(nil)
	require.NoError(t, err)

	t.Run("can push", func(t *testing.T) {
		require.NoError(t, err)
	})

	t.Run("can check if remote branch exists -- positive result", func(t *testing.T) {
		var exists bool
		exists, err = rep.RemoteBranchExists("main")
		require.NoError(t, err)
		require.True(t, exists)
	})

	testBranch := fmt.Sprintf("test-branch-%s", uuid.NewString())
	err = rep.CreateChildBranch(testBranch)
	require.NoError(t, err)

	t.Run("can create a child branch", func(t *testing.T) {
		require.NoError(t, err)
	})

	err = os.WriteFile(fmt.Sprintf("%s/%s", rep.Dir(), "test.txt"), []byte("bar"), 0600)
	require.NoError(t, err)

	t.Run("can hard reset", func(t *testing.T) {
		var hasDiffs bool
		hasDiffs, err = rep.HasDiffs()
		require.NoError(t, err)
		require.True(t, hasDiffs)
		err = rep.ResetHard()
		require.NoError(t, err)
		hasDiffs, err = rep.HasDiffs()
		require.NoError(t, err)
		require.False(t, hasDiffs)
	})

	t.Run("can create an orphaned branch", func(t *testing.T) {
		testBranch := fmt.Sprintf("test-branch-%s", uuid.NewString())
		err = rep.CreateOrphanedBranch(testBranch)
		require.NoError(t, err)
	})

	t.Run("can load an existing repo", func(t *testing.T) {
		existingRepo, err := LoadRepo(
			rep.Dir(),
			&LoadRepoOptions{
				Credentials: &testRepoCreds,
			},
		)
		require.NoError(t, err)
		require.Equal(t, rep, existingRepo)
	})

	t.Run("can close repo", func(t *testing.T) {
		require.NoError(t, rep.Close())
		_, err := os.Stat(r.HomeDir())
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
	})

}

func TestGetDiffPathsForMergeCommit(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(t)
	defer testServer.Close()

	rep, err := Clone(
		testRepoURL,
		&ClientOptions{Credentials: &testRepoCreds},
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, rep)
	defer rep.Close()

	wt := internalWorkTree(t, rep)

	// Create initial commit on main with base files
	require.NoError(t, os.MkdirAll(fmt.Sprintf("%s/foo", rep.Dir()), 0o755))
	require.NoError(
		t,
		os.WriteFile(
			fmt.Sprintf("%s/foo/file1.txt", rep.Dir()),
			[]byte("base"),
			0o600,
		),
	)
	require.NoError(
		t,
		os.WriteFile(
			fmt.Sprintf("%s/foo/file2.txt", rep.Dir()),
			[]byte("base"),
			0o600,
		),
	)
	require.NoError(t, rep.AddAllAndCommit("initial commit", nil))

	// Create branch-a and modify file1
	require.NoError(t, rep.CreateChildBranch("branch-a"))
	require.NoError(
		t,
		os.WriteFile(
			fmt.Sprintf("%s/foo/file1.txt", rep.Dir()),
			[]byte("changed by branch-a"),
			0o600,
		),
	)
	require.NoError(t, rep.AddAllAndCommit("branch-a: modify file1", nil))

	// Back to main, create branch-b, modify file2
	require.NoError(t, rep.Checkout("main"))
	require.NoError(t, rep.CreateChildBranch("branch-b"))
	require.NoError(
		t,
		os.WriteFile(
			fmt.Sprintf("%s/foo/file2.txt", rep.Dir()),
			[]byte("changed by branch-b"),
			0o600,
		),
	)
	require.NoError(t, rep.AddAllAndCommit("branch-b: modify file2", nil))

	// Merge branch-b into main
	require.NoError(t, rep.Checkout("main"))
	_, err = libExec.Exec(wt.buildGitCommand(
		"merge", "branch-b", "--no-ff", "-m", "merge branch-b",
	))
	require.NoError(t, err)

	// Merge branch-a into main
	_, err = libExec.Exec(wt.buildGitCommand(
		"merge", "branch-a", "--no-ff", "-m", "merge branch-a",
	))
	require.NoError(t, err)

	mergeCommitID, err := rep.LastCommitID()
	require.NoError(t, err)

	// GetDiffPathsForCommitID on the merge commit should return only
	// the file introduced by that merge (file1, from branch-a), not
	// file2 which was already on main via the earlier merge of branch-b.
	paths, err := rep.GetDiffPathsForCommitID(mergeCommitID)
	require.NoError(t, err)
	require.Equal(t, []string{"foo/file1.txt"}, paths)
}
