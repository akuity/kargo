package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetLatestCommitID(t *testing.T) {
	testCases := []struct {
		name       string
		repoURL    string
		branch     string
		assertions func(string, error)
	}{
		{
			name:    "error cloning repo",
			repoURL: "fake-url", // This should force a failure
			assertions: func(_ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error cloning git repo")
			},
		},

		{
			name:    "error checking out branch",
			repoURL: "https://github.com/argoproj/argo-cd.git",
			branch:  "bogus", // This should force a failure
			assertions: func(_ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error checking out branch")
			},
		},

		{
			name:    "success",
			repoURL: "https://github.com/argoproj/argo-cd.git",
			assertions: func(commit string, err error) {
				require.NoError(t, err)
				require.NotEmpty(t, commit)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				GetLatestCommitID(testCase.repoURL, testCase.branch, nil),
			)
		})
	}
}

func TestApplyUpdate(t *testing.T) {
	testCases := []struct {
		name       string
		repoURL    string
		branch     string
		updateFn   func(string, string) (string, error)
		assertions func(string, error)
	}{
		{
			name:    "error cloning repo",
			repoURL: "fake-url", // This should force a failure
			assertions: func(_ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error cloning git repo")
			},
		},

		{
			name:    "error checking out branch",
			repoURL: "https://github.com/argoproj/argo-cd.git",
			branch:  "bogus", // This should force a failure
			assertions: func(_ string, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error checking out branch")
			},
		},

		{
			name:    "error running update function",
			repoURL: "https://github.com/argoproj/argo-cd.git",
			updateFn: func(string, string) (string, error) {
				return "", errors.New("something went wrong")
			},
			assertions: func(_ string, err error) {
				require.Error(t, err)
				require.Equal(t, err.Error(), "something went wrong")
			},
		},

		// TODO: Hard to test success case without actually pushing to a remote
		// repository.
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				ApplyUpdate(
					testCase.repoURL,
					testCase.branch,
					testCase.branch,
					nil,
					testCase.updateFn,
				),
			)
		})
	}
}

func TestMoveRepoContents(t *testing.T) {
	const subdirCount = 50
	const fileCount = 50
	// Create dummy repo dir
	srcDir, err := createDummyRepoDir(subdirCount, fileCount)
	defer os.RemoveAll(srcDir)
	require.NoError(t, err)
	// Double-check the setup
	dirEntries, err := os.ReadDir(srcDir)
	require.NoError(t, err)
	require.Len(t, dirEntries, subdirCount+fileCount+1)
	// Create destination dir
	destDir, err := os.MkdirTemp("", "")
	defer os.RemoveAll(destDir)
	require.NoError(t, err)
	// Move
	err = moveRepoContents(srcDir, destDir)
	require.NoError(t, err)
	// .git should not have moved
	_, err = os.Stat(filepath.Join(srcDir, ".git"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(destDir, ".git"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
	// Everything else should have moved
	dirEntries, err = os.ReadDir(srcDir)
	require.NoError(t, err)
	require.Len(t, dirEntries, 1)
	dirEntries, err = os.ReadDir(destDir)
	require.NoError(t, err)
	require.Len(t, dirEntries, subdirCount+fileCount)
}

func TestDeleteRepoContents(t *testing.T) {
	const subdirCount = 50
	const fileCount = 50
	// Create dummy repo dir
	dir, err := createDummyRepoDir(subdirCount, fileCount)
	defer os.RemoveAll(dir)
	require.NoError(t, err)
	// Double-check the setup
	dirEntries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, dirEntries, subdirCount+fileCount+1)
	// Delete
	err = deleteRepoContents(dir)
	require.NoError(t, err)
	// .git should not have been deleted
	_, err = os.Stat(filepath.Join(dir, ".git"))
	require.NoError(t, err)
	// Everything else should be deleted
	dirEntries, err = os.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, dirEntries, 1)
}

func createDummyRepoDir(dirCount, fileCount int) (string, error) {
	// Create a directory
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		return dir, err
	}
	// Add a dummy .git/ subdir
	if err = os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		return dir, err
	}
	// Add some dummy dirs
	for i := 0; i < dirCount; i++ {
		if err = os.Mkdir(filepath.Join(dir, fmt.Sprintf("dir-%d", i)), 0755); err != nil {
			return dir, err
		}
	}
	// Add some dummy files
	for i := 0; i < fileCount; i++ {
		file, err := os.Create(filepath.Join(dir, fmt.Sprintf("file-%d", i)))
		if err != nil {
			return dir, err
		}
		if err = file.Close(); err != nil {
			return dir, err
		}
	}
	return dir, nil
}
