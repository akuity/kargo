package promotion

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/akuity/bookkeeper/pkg/git"
	api "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/credentials"
)

func TestNewGitMechanism(t *testing.T) {
	pm := newGitMechanism(
		"fake-name",
		&credentials.FakeDB{},
		func([]api.GitRepoUpdate) []api.GitRepoUpdate {
			return nil
		},
		func(
			update api.GitRepoUpdate,
			newState api.StageState,
			homeDir string,
			workingDir string,
		) ([]string, error) {
			return nil, nil
		},
	)
	gpm, ok := pm.(*gitMechanism)
	require.True(t, ok)
	require.NotEmpty(t, gpm.name)
	require.NotNil(t, gpm.selectUpdatesFn)
	require.NotNil(t, gpm.doSingleUpdateFn)
	require.NotNil(t, gpm.getReadRefFn)
	require.NotNil(t, gpm.getCredentialsFn)
	require.NotNil(t, gpm.gitCommitFn)
	require.NotNil(t, gpm.applyConfigManagementFn)
}

func TestGitGetName(t *testing.T) {
	const testName = "fake name"
	pm := newGitMechanism(testName, nil, nil, nil)
	require.Equal(t, testName, pm.GetName())
}

func TestGitPromote(t *testing.T) {
	testCases := []struct {
		name       string
		promoMech  *gitMechanism
		assertions func(newStateIn, newStateOut api.StageState, err error)
	}{
		{
			name: "no updates",
			promoMech: &gitMechanism{
				selectUpdatesFn: func([]api.GitRepoUpdate) []api.GitRepoUpdate {
					return nil
				},
			},
			assertions: func(newStateIn, newStateOut api.StageState, err error) {
				require.NoError(t, err)
				require.Equal(t, newStateIn, newStateOut)
			},
		},
		{
			name: "error applying single update",
			promoMech: &gitMechanism{
				selectUpdatesFn: func([]api.GitRepoUpdate) []api.GitRepoUpdate {
					return []api.GitRepoUpdate{{}}
				},
				doSingleUpdateFn: func(
					_ context.Context,
					_ string,
					_ api.GitRepoUpdate,
					newState api.StageState,
				) (api.StageState, error) {
					return newState, errors.New("something went wrong")
				},
			},
			assertions: func(newStateIn, newStateOut api.StageState, err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newStateIn, newStateOut)
			},
		},
		{
			name: "success",
			promoMech: &gitMechanism{
				selectUpdatesFn: func([]api.GitRepoUpdate) []api.GitRepoUpdate {
					return []api.GitRepoUpdate{{}}
				},
				doSingleUpdateFn: func(
					_ context.Context,
					_ string,
					_ api.GitRepoUpdate,
					newState api.StageState,
				) (api.StageState, error) {
					return newState, nil
				},
			},
			assertions: func(newStateIn, newStateOut api.StageState, err error) {
				require.NoError(t, err)
				require.Equal(t, newStateIn, newStateOut)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			newStateIn := api.StageState{}
			newStateOut, err := testCase.promoMech.Promote(
				context.Background(),
				&api.Stage{
					Spec: &api.StageSpec{
						PromotionMechanisms: &api.PromotionMechanisms{},
					},
				},
				newStateIn,
			)
			testCase.assertions(newStateIn, newStateOut, err)
		})
	}
}

func TestGitDoSingleUpdate(t *testing.T) {
	const testRef = "fake-ref"
	testCases := []struct {
		name       string
		promoMech  *gitMechanism
		assertions func(newStateIn, newStateOut api.StageState, err error)
	}{
		{
			name: "error getting readref",
			promoMech: &gitMechanism{
				getReadRefFn: func(
					api.GitRepoUpdate,
					[]api.GitCommit,
				) (string, int, error) {
					return "", 0, errors.New("something went wrong")
				},
			},
			assertions: func(newStateIn, newStateOut api.StageState, err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newStateIn, newStateOut)
			},
		},
		{
			name: "error getting repo credentials",
			promoMech: &gitMechanism{
				getReadRefFn: func(
					api.GitRepoUpdate,
					[]api.GitCommit,
				) (string, int, error) {
					return testRef, 0, nil
				},
				getCredentialsFn: func(
					context.Context,
					string,
					string,
				) (*git.RepoCredentials, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(newStateIn, newStateOut api.StageState, err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newStateIn, newStateOut)
			},
		},
		{
			name: "error committing change to repo",
			promoMech: &gitMechanism{
				getReadRefFn: func(
					api.GitRepoUpdate,
					[]api.GitCommit,
				) (string, int, error) {
					return testRef, 0, nil
				},
				getCredentialsFn: func(
					context.Context,
					string,
					string,
				) (*git.RepoCredentials, error) {
					return nil, nil
				},
				gitCommitFn: func(
					update api.GitRepoUpdate,
					newState api.StageState,
					readRef string,
					writeBranch string,
					creds *git.RepoCredentials,
				) (string, error) {
					return "", errors.New("something went wrong")
				},
			},
			assertions: func(newStateIn, newStateOut api.StageState, err error) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newStateIn, newStateOut)
			},
		},
		{
			name: "success",
			promoMech: &gitMechanism{
				getReadRefFn: func(
					api.GitRepoUpdate,
					[]api.GitCommit,
				) (string, int, error) {
					return testRef, 0, nil
				},
				getCredentialsFn: func(
					context.Context,
					string,
					string,
				) (*git.RepoCredentials, error) {
					return nil, nil
				},
				gitCommitFn: func(
					update api.GitRepoUpdate,
					newState api.StageState,
					readRef string,
					writeBranch string,
					creds *git.RepoCredentials,
				) (string, error) {
					return "fake-commit-id", nil
				},
			},
			assertions: func(newStateIn, newStateOut api.StageState, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					"fake-commit-id",
					newStateOut.Commits[0].HealthCheckCommit,
				)
				// The newState is otherwise unaltered
				newStateIn.Commits[0].HealthCheckCommit = ""
				require.Equal(t, newStateIn, newStateOut)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			newStateIn := api.StageState{
				Commits: []api.GitCommit{{}},
			}
			newStateOut, err := testCase.promoMech.doSingleUpdate(
				context.Background(),
				"fake-namespace",
				api.GitRepoUpdate{},
				newStateIn,
			)
			testCase.assertions(newStateIn, newStateOut, err)
		})
	}
}

func TestGetReadRef(t *testing.T) {
	const testBranch = "fake-branch"
	testCases := []struct {
		name       string
		update     api.GitRepoUpdate
		commits    []api.GitCommit
		assertions func(readBranch string, commitIndex int, err error)
	}{
		{
			name: "update's RepoURL does not match any subscription",
			update: api.GitRepoUpdate{
				RepoURL:    "fake-url",
				ReadBranch: testBranch,
			},
			assertions: func(readBranch string, commitIndex int, err error) {
				require.NoError(t, err)
				require.Equal(t, testBranch, readBranch)
				require.Equal(t, -1, commitIndex)
			},
		},
		{
			name: "subscription-loop avoided",
			update: api.GitRepoUpdate{
				RepoURL:     "fake-url",
				WriteBranch: testBranch,
			},
			commits: []api.GitCommit{
				{
					RepoURL: "fake-url",
					Branch:  testBranch,
				},
			},
			assertions: func(_ string, _ int, err error) {
				require.Error(t, err)
				require.Contains(
					t,
					err.Error(),
					"because it will form a subscription loop",
				)
			},
		},
		{
			name: "success",
			update: api.GitRepoUpdate{
				RepoURL: "fake-url",
			},
			commits: []api.GitCommit{
				{
					RepoURL: "fake-url",
					ID:      "fake-commit-id",
					Branch:  testBranch,
				},
			},
			assertions: func(readBranch string, commitIndex int, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-commit-id", readBranch)
				require.Equal(t, 0, commitIndex)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				getReadRef(testCase.update, testCase.commits),
			)
		})
	}
}

func TestGetRepoCredentials(t *testing.T) {
	testCases := []struct {
		name          string
		credentialsDB credentials.Database
		assertions    func(*git.RepoCredentials, error)
	}{
		{
			name: "error getting credentials from database",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{},
						false, errors.New("something went wrong")
				},
			},
			assertions: func(_ *git.RepoCredentials, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "error obtaining credentials")
				require.Contains(t, err.Error(), "something went wrong")
			},
		},
		{
			name: "no credentials found in database",
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
			assertions: func(creds *git.RepoCredentials, err error) {
				require.NoError(t, err)
				require.Nil(t, creds)
			},
		},
		{
			name: "credentials found in database",
			credentialsDB: &credentials.FakeDB{
				GetFn: func(
					context.Context,
					string,
					credentials.Type,
					string,
				) (credentials.Credentials, bool, error) {
					return credentials.Credentials{
						Username: "fake-username",
						Password: "fake-password",
					}, true, nil
				},
			},
			assertions: func(creds *git.RepoCredentials, err error) {
				require.NoError(t, err)
				require.Equal(
					t,
					&git.RepoCredentials{
						Username: "fake-username",
						Password: "fake-password",
					},
					creds,
				)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(
				getRepoCredentialsFn(testCase.credentialsDB)(
					context.Background(),
					"fake-namespace",
					"fake-repo-url",
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
