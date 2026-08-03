package git

import (
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRepo(t *testing.T) {
	testServer, testRepoURL, testRepoCreds := setupRemoteRepo(t)
	defer testServer.Close()

	rep, err := Clone(
		t.Context(),
		testRepoURL,
		&ClientOptions{Credentials: &testRepoCreds},
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, rep)
	defer rep.Close(t.Context())
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
		hasDiffs, err = rep.HasDiffs(t.Context())
		require.NoError(t, err)
		require.False(t, hasDiffs)
	})

	err = os.WriteFile(fmt.Sprintf("%s/%s", rep.Dir(), "test.txt"), []byte("foo"), 0600)
	require.NoError(t, err)

	t.Run("can check for diffs -- positive result", func(t *testing.T) {
		var hasDiffs bool
		hasDiffs, err = rep.HasDiffs(t.Context())
		require.NoError(t, err)
		require.True(t, hasDiffs)
	})

	testCommitMessage := fmt.Sprintf(`test commit %s

with a body
`, uuid.NewString())
	err = rep.AddAllAndCommit(t.Context(), testCommitMessage, nil)
	require.NoError(t, err)

	t.Run("can commit", func(t *testing.T) {
		require.NoError(t, err)
	})

	lastCommitID, err := rep.LastCommitID(t.Context())
	require.NoError(t, err)

	t.Run("can get last commit id", func(t *testing.T) {
		require.NoError(t, err)
		require.NotEmpty(t, lastCommitID)
	})

	t.Run("can get commit message by id", func(t *testing.T) {
		var msg string
		msg, err = rep.CommitMessage(t.Context(), lastCommitID)
		require.NoError(t, err)
		require.Equal(t, testCommitMessage, msg)
	})

	t.Run("can get diff paths", func(t *testing.T) {
		var paths []string
		paths, err = rep.GetDiffPathsForCommitID(t.Context(), lastCommitID)
		require.NoError(t, err)
		require.Len(t, paths, 1)
	})

	t.Run("can check if remote branch exists -- negative result", func(t *testing.T) {
		var exists bool
		exists, err = rep.RemoteBranchExists(t.Context(), "main") // The remote repo is empty!
		require.NoError(t, err)
		require.False(t, exists)
	})

	err = rep.Push(t.Context(), nil)
	require.NoError(t, err)

	t.Run("can push", func(t *testing.T) {
		require.NoError(t, err)
	})

	t.Run("can check if remote branch exists -- positive result", func(t *testing.T) {
		var exists bool
		exists, err = rep.RemoteBranchExists(t.Context(), "main")
		require.NoError(t, err)
		require.True(t, exists)
	})

	testBranch := fmt.Sprintf("test-branch-%s", uuid.NewString())
	err = rep.CreateChildBranch(t.Context(), testBranch)
	require.NoError(t, err)

	t.Run("can create a child branch", func(t *testing.T) {
		require.NoError(t, err)
	})

	err = os.WriteFile(fmt.Sprintf("%s/%s", rep.Dir(), "test.txt"), []byte("bar"), 0600)
	require.NoError(t, err)

	t.Run("can hard reset", func(t *testing.T) {
		var hasDiffs bool
		hasDiffs, err = rep.HasDiffs(t.Context())
		require.NoError(t, err)
		require.True(t, hasDiffs)
		err = rep.ResetHard(t.Context())
		require.NoError(t, err)
		hasDiffs, err = rep.HasDiffs(t.Context())
		require.NoError(t, err)
		require.False(t, hasDiffs)
	})

	t.Run("can create an orphaned branch", func(t *testing.T) {
		testBranch := fmt.Sprintf("test-branch-%s", uuid.NewString())
		err = rep.CreateOrphanedBranch(t.Context(), testBranch)
		require.NoError(t, err)
	})

	t.Run("can load an existing repo", func(t *testing.T) {
		existingRepo, err := LoadRepo(
			t.Context(),
			rep.Dir(),
			&LoadRepoOptions{Credentials: &testRepoCreds},
		)
		require.NoError(t, err)
		require.Equal(t, rep, existingRepo)
	})

	t.Run("can close repo", func(t *testing.T) {
		require.NoError(t, rep.Close(t.Context()))
		_, err := os.Stat(r.HomeDir())
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
	})

}
