package promotions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/akuity/bookkeeper/pkg/git"
	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

func TestApplyGitRepoUpdate(t *testing.T) {
	testCases := []struct {
		name             string
		newState         api.EnvironmentState
		credentialsDB    credentials.Database
		gitApplyUpdateFn func(
			string,
			string,
			string,
			*git.RepoCredentials,
			func(homeDir, workingDir string) (string, error),
		) (string, error)
		assertions func(inState, outState api.EnvironmentState, err error)
	}{
		{
			name: "invalid update",
			newState: api.EnvironmentState{
				Commits: []api.GitCommit{
					{
						RepoURL: "fake-url",
						Branch:  "fake-branch",
					},
				},
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"invalid update specified; cannot write to branch",
				)
				require.Contains(
					t,
					err.Error(),
					"because it will form a subscription loop",
				)
			},
		},

		{
			name: "error getting repo credentials",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false,
						errors.New("something went wrong")
				},
			},
			assertions: func(_, _ api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"error obtaining credentials for git repo",
				)
				require.Contains(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "error applying updates",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, nil
				},
			},
			gitApplyUpdateFn: func(
				string,
				string,
				string,
				*git.RepoCredentials,
				func(string, string) (string, error),
			) (string, error) {
				return "", errors.New("something went wrong")
			},
			assertions: func(_, _ api.EnvironmentState, err error) {
				require.Error(t, err)
				require.Equal(t, err.Error(), "something went wrong")
			},
		},

		{
			name: "success",
			newState: api.EnvironmentState{
				Commits: []api.GitCommit{
					{
						RepoURL: "fake-url",
						// This branch deliberately doesn't match the branch we read from
						Branch: "another-fake-branch",
					},
				},
			},
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{}, false, nil
				},
			},
			gitApplyUpdateFn: func(
				string,
				string,
				string,
				*git.RepoCredentials,
				func(string, string) (string, error),
			) (string, error) {
				return "new-fake-commit", nil
			},
			assertions: func(inState, outState api.EnvironmentState, err error) {
				require.NoError(t, err)
				require.Len(t, outState.Commits, 1)
				// Check that HealthCheckCommit got set
				require.NotEmpty(t, outState.Commits[0].HealthCheckCommit)
				// Everything else should be unchanged
				outState.Commits[0].HealthCheckCommit = ""
				outState.ID = inState.ID
				require.Equal(t, inState, outState)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			r := reconciler{
				credentialsDB:    testCase.credentialsDB,
				gitApplyUpdateFn: testCase.gitApplyUpdateFn,
			}
			outState, err := r.applyGitRepoUpdate(
				context.Background(),
				"fake-namespace",
				testCase.newState,
				api.GitRepoUpdate{
					RepoURL:     "fake-url",
					WriteBranch: "fake-branch",
				},
			)
			testCase.assertions(testCase.newState, outState, err)
		})
	}
}

func TestGitApplyUpdate(t *testing.T) {
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
				gitApplyUpdate(
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
		if err = os.Mkdir(
			filepath.Join(dir, fmt.Sprintf("dir-%d", i)),
			0755,
		); err != nil {
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

func TestBuildCommitMessage(t *testing.T) {
	testCases := []struct {
		name          string
		changeSummary []string
		assertions    func(msg string)
	}{
		{
			// This shouldn't really happen, but we're careful to handle it anyway,
			// so we might as well test it.
			name:          "nil change summary",
			changeSummary: nil,
			assertions: func(msg string) {
				require.Equal(t, "Kargo applied some changes", msg)
			},
		},
		{
			// This shouldn't really happen, but we're careful to handle it anyway,
			// so we might as well test it.
			name:          "empty change summary",
			changeSummary: []string{},
			assertions: func(msg string) {
				require.Equal(t, "Kargo applied some changes", msg)
			},
		},
		{
			name: "change summary contains one item",
			changeSummary: []string{
				"fake-change",
			},
			assertions: func(msg string) {
				require.Equal(t, "fake-change", msg)
			},
		},
		{
			name: "change summary contains multiple items",
			changeSummary: []string{
				"fake-change",
				"another-fake-change",
			},
			assertions: func(msg string) {
				require.Equal(
					t,
					[]string{
						"Kargo applied multiple changes",
						"",
						"Including:",
						"",
						"  * fake-change",
						"  * another-fake-change",
					},
					strings.Split(msg, "\n"),
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(buildCommitMessage(testCase.changeSummary))
		})
	}
}
