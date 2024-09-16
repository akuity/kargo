package git

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/internal/types"
)

func TestRepo(t *testing.T) {
	testRepoCreds := RepoCredentials{
		Username: "fake-username",
		Password: "fake-password",
	}

	// This will be something to opt into because on some OSes, this will lead
	// to keychain-related prompts.
	var useAuth bool
	if useAuthStr := os.Getenv("TEST_GIT_CLIENT_WITH_AUTH"); useAuthStr != "" {
		useAuth = types.MustParseBool(useAuthStr)
	}
	service := gitkit.New(
		gitkit.Config{
			Dir:        t.TempDir(),
			AutoCreate: true,
			Auth:       useAuth,
		},
	)
	require.NoError(t, service.Setup())
	service.AuthFunc =
		func(cred gitkit.Credential, _ *gitkit.Request) (bool, error) {
			return cred.Username == testRepoCreds.Username &&
				cred.Password == testRepoCreds.Password, nil
		}
	server := httptest.NewServer(service)
	defer server.Close()

	testRepoURL := fmt.Sprintf("%s/test.git", server.URL)

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
		var repoURL *url.URL
		repoURL, err = url.Parse(r.url)
		require.NoError(t, err)
		repoURL.User = nil
		require.Equal(t, testRepoURL, repoURL.String())
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
		require.Equal(t, r.url, r.URL())
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

	testCommitMessage := fmt.Sprintf("test commit %s", uuid.NewString())
	err = rep.AddAllAndCommit(testCommitMessage)
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
		// "master" is still the default branch name for a new repository unless
		// you configure it otherwise.
		exists, err = rep.RemoteBranchExists("master")
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
