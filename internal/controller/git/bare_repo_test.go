package git

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/internal/types"
)

func TestBareRepo(t *testing.T) {
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

	setupRep, err := Clone(
		testRepoURL,
		&ClientOptions{
			Credentials: &testRepoCreds,
		},
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, setupRep)
	defer setupRep.Close()
	err = os.WriteFile(fmt.Sprintf("%s/%s", setupRep.Dir(), "test.txt"), []byte("foo"), 0600)
	require.NoError(t, err)
	err = setupRep.AddAllAndCommit(fmt.Sprintf("initial commit %s", uuid.NewString()))
	require.NoError(t, err)
	err = setupRep.Push(nil)
	require.NoError(t, err)
	err = setupRep.Close()
	require.NoError(t, err)

	rep, err := CloneBare(
		testRepoURL,
		&ClientOptions{
			Credentials: &testRepoCreds,
		},
		nil,
	)
	require.NoError(t, err)
	require.NotNil(t, rep)
	defer rep.Close()
	r, ok := rep.(*bareRepo)
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

	workingTreePath := filepath.Join(rep.HomeDir(), "working-tree")
	workTree, err := rep.AddWorkTree(
		workingTreePath,
		// "master" is still the default branch name for a new repository unless
		// you configure it otherwise.
		&AddWorkTreeOptions{Ref: "master"},
	)

	require.NoError(t, err)
	defer workTree.Close()

	t.Run("add a working tree", func(t *testing.T) {
		require.NoError(t, err)
		require.Equal(t, workingTreePath, workTree.Dir())
		_, err = os.Stat(workTree.Dir())
		require.NoError(t, err)
	})

	t.Run("can list working trees", func(t *testing.T) {
		var workTrees []WorkTree
		workTrees, err = rep.WorkTrees()
		require.NoError(t, err)
		require.Len(t, workTrees, 1)
		require.Equal(t, workTree, workTrees[0])
	})

	t.Run("can remove a working tree", func(t *testing.T) {
		err = rep.RemoveWorkTree(workTree.Dir())
		require.NoError(t, err)
		trees, err := rep.WorkTrees()
		require.NoError(t, err)
		require.Len(t, trees, 0)
		_, err = os.Stat(workTree.Dir())
		require.True(t, os.IsNotExist(err))
	})

	t.Run("can load an existing repo", func(t *testing.T) {
		existingRepo, err := LoadBareRepo(
			rep.Dir(),
			&LoadBareRepoOptions{
				Credentials: &testRepoCreds,
			},
		)
		require.NoError(t, err)
		require.Equal(t, rep, existingRepo)
	})

	t.Run("can close repo", func(t *testing.T) {
		require.NoError(t, rep.Close())
		_, err := os.Stat(rep.HomeDir())
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
	})

}
