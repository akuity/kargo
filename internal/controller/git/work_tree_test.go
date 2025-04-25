package git

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/sosedoff/gitkit"
	"github.com/stretchr/testify/require"

	"github.com/akuity/kargo/internal/types"
)

func TestWorkTree(t *testing.T) {
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
	err = setupRep.AddAllAndCommit(fmt.Sprintf("initial commit %s", uuid.NewString()), nil)
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

	workingTreePath := filepath.Join(rep.HomeDir(), "working-tree")
	workTree, err := rep.AddWorkTree(
		workingTreePath,
		// "master" is still the default branch name for a new repository unless
		// you configure it otherwise.
		&AddWorkTreeOptions{Ref: "master"},
	)
	require.NoError(t, err)
	defer workTree.Close()

	t.Run("can load an existing working tree", func(t *testing.T) {
		existingWorkTree, err := LoadWorkTree(
			workTree.Dir(),
			&LoadWorkTreeOptions{
				Credentials: &testRepoCreds,
			},
		)
		require.NoError(t, err)
		require.Equal(t, workTree, existingWorkTree)
	})

	t.Run("can close working tree", func(t *testing.T) {
		require.NoError(t, workTree.Close())
		_, err := os.Stat(workTree.Dir())
		require.Error(t, err)
		require.True(t, os.IsNotExist(err))
	})

}
