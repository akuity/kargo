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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	"github.com/akuity/kargo/internal/credentials"
	libGit "github.com/akuity/kargo/internal/git"
)

func TestNewGitMechanism(t *testing.T) {
	pm := newGitMechanism(
		"fake-name",
		&credentials.FakeDB{},
		func([]kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate {
			return nil
		},
		func(
			context.Context,
			kargoapi.GitRepoUpdate,
			kargoapi.FreightReference,
			string, string, string, string,
			git.RepoCredentials,
		) ([]string, error) {
			return nil, nil
		},
	)
	gpm, ok := pm.(*gitMechanism)
	require.True(t, ok)
	require.NotEmpty(t, gpm.name)
	require.NotNil(t, gpm.selectUpdatesFn)
	require.NotNil(t, gpm.cloneFreightCommitsFn)
	require.NotNil(t, gpm.doSingleUpdateFn)
	require.NotNil(t, gpm.getReadRefFn)
	require.NotNil(t, gpm.getCredentialsFn)
	require.NotNil(t, gpm.getAuthorFn)
	require.NotNil(t, gpm.gitCommitFn)
	require.NotNil(t, gpm.applyCopyPatchesFn)
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
		freight    kargoapi.FreightReference
		promoMech  *gitMechanism
		assertions func(
			t *testing.T,
			status *kargoapi.PromotionStatus,
			newFreightIn kargoapi.FreightReference,
			newFreightOut kargoapi.FreightReference,
			err error,
		)
	}{
		{
			name: "no updates",
			promoMech: &gitMechanism{
				selectUpdatesFn: func([]kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate {
					return nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "error cloning freight commits",
			freight: kargoapi.FreightReference{
				Commits: []kargoapi.GitCommit{{
					RepoURL: "https://example.com/repo.git",
				}},
			},
			promoMech: &gitMechanism{
				selectUpdatesFn: func([]kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate {
					return []kargoapi.GitRepoUpdate{
						{
							Patches: []kargoapi.PatchOperation{
								{
									Copy: &kargoapi.CopyPatchOperation{
										RepoURL: "https://example.com/repo.git",
									},
								},
							},
						},
					}
				},
				cloneFreightCommitsFn: func(
					_ context.Context,
					_ string,
					_ []kargoapi.GitCommit,
				) (gitRepositories, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "error applying single update",
			promoMech: &gitMechanism{
				selectUpdatesFn: func([]kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate {
					return []kargoapi.GitRepoUpdate{{}}
				},
				doSingleUpdateFn: func(
					_ context.Context,
					_ *kargoapi.Promotion,
					_ kargoapi.GitRepoUpdate,
					newFreight kargoapi.FreightReference,
					_ gitRepositories,
				) (*kargoapi.PromotionStatus, kargoapi.FreightReference, error) {
					return nil, newFreight, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "success",
			promoMech: &gitMechanism{
				selectUpdatesFn: func([]kargoapi.GitRepoUpdate) []kargoapi.GitRepoUpdate {
					return []kargoapi.GitRepoUpdate{{}}
				},
				doSingleUpdateFn: func(
					_ context.Context,
					_ *kargoapi.Promotion,
					_ kargoapi.GitRepoUpdate,
					newFreight kargoapi.FreightReference,
					_ gitRepositories,
				) (*kargoapi.PromotionStatus, kargoapi.FreightReference, error) {
					return &kargoapi.PromotionStatus{Phase: kargoapi.PromotionPhaseSucceeded}, newFreight, nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			newFreightIn := testCase.freight
			status, newFreightOut, err := testCase.promoMech.Promote(
				context.Background(),
				&kargoapi.Stage{
					Spec: kargoapi.StageSpec{
						PromotionMechanisms: &kargoapi.PromotionMechanisms{},
					},
				},
				&kargoapi.Promotion{},
				newFreightIn,
			)
			testCase.assertions(t, status, newFreightIn, newFreightOut, err)
		})
	}
}

func TestGitDoSingleUpdate(t *testing.T) {
	const testRef = "fake-ref"
	testCases := []struct {
		name       string
		promoMech  *gitMechanism
		assertions func(
			t *testing.T,
			status *kargoapi.PromotionStatus,
			newFreightIn kargoapi.FreightReference,
			newFreightOut kargoapi.FreightReference,
			err error,
		)
	}{
		{
			name: "error getting readref",
			promoMech: &gitMechanism{
				getReadRefFn: func(
					kargoapi.GitRepoUpdate,
					[]kargoapi.GitCommit,
				) (string, int, error) {
					return "", 0, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "error getting repo credentials",
			promoMech: &gitMechanism{
				getReadRefFn: func(
					kargoapi.GitRepoUpdate,
					[]kargoapi.GitCommit,
				) (string, int, error) {
					return testRef, 0, nil
				},
				getAuthorFn: func() (*git.User, error) {
					return nil, nil
				},
				getCredentialsFn: func(
					context.Context,
					string,
					string,
				) (*git.RepoCredentials, error) {
					return nil, errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "error getting author",
			promoMech: &gitMechanism{
				getReadRefFn: func(
					kargoapi.GitRepoUpdate,
					[]kargoapi.GitCommit,
				) (string, int, error) {
					return testRef, 0, nil
				},
				getAuthorFn: func() (*git.User, error) {
					return nil, errors.New("something went wrong")
				},
				getCredentialsFn: func(
					context.Context,
					string,
					string,
				) (*git.RepoCredentials, error) {
					return nil, nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "error committing change to repo",
			promoMech: &gitMechanism{
				getReadRefFn: func(
					kargoapi.GitRepoUpdate,
					[]kargoapi.GitCommit,
				) (string, int, error) {
					return testRef, 0, nil
				},
				getAuthorFn: func() (*git.User, error) {
					return nil, nil
				},
				getCredentialsFn: func(
					context.Context,
					string,
					string,
				) (*git.RepoCredentials, error) {
					return nil, nil
				},
				gitCommitFn: func(
					context.Context,
					kargoapi.GitRepoUpdate,
					kargoapi.FreightReference,
					string,
					string,
					string,
					git.Repo,
					git.RepoCredentials,
					gitRepositories,
				) (string, error) {
					return "", errors.New("something went wrong")
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.Error(t, err)
				require.Equal(t, "something went wrong", err.Error())
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
		{
			name: "success",
			promoMech: &gitMechanism{
				getReadRefFn: func(
					kargoapi.GitRepoUpdate,
					[]kargoapi.GitCommit,
				) (string, int, error) {
					return testRef, 0, nil
				},
				getAuthorFn: func() (*git.User, error) {
					return nil, nil
				},
				getCredentialsFn: func(
					context.Context,
					string,
					string,
				) (*git.RepoCredentials, error) {
					return nil, nil
				},
				gitCommitFn: func(
					context.Context,
					kargoapi.GitRepoUpdate,
					kargoapi.FreightReference,
					string,
					string,
					string,
					git.Repo,
					git.RepoCredentials,
					gitRepositories,
				) (string, error) {
					return "fake-commit-id", nil
				},
			},
			assertions: func(
				t *testing.T,
				_ *kargoapi.PromotionStatus,
				newFreightIn kargoapi.FreightReference,
				newFreightOut kargoapi.FreightReference,
				err error,
			) {
				require.NoError(t, err)
				require.Equal(
					t,
					"fake-commit-id",
					newFreightOut.Commits[0].HealthCheckCommit,
				)
				// The newFreight is otherwise unaltered
				newFreightIn.Commits[0].HealthCheckCommit = ""
				require.Equal(t, newFreightIn, newFreightOut)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			newFreightIn := kargoapi.FreightReference{
				Commits: []kargoapi.GitCommit{{}},
			}
			status, newFreightOut, err := testCase.promoMech.doSingleUpdate(
				context.Background(),
				&kargoapi.Promotion{
					ObjectMeta: metav1.ObjectMeta{Namespace: "fake-namespace"},
				},
				kargoapi.GitRepoUpdate{RepoURL: "https://github.com/akuity/kargo"},
				newFreightIn,
				nil,
			)
			testCase.assertions(t, status, newFreightIn, newFreightOut, err)
		})
	}
}

func TestGetReadRef(t *testing.T) {
	const testBranch = "fake-branch"
	testCases := []struct {
		name       string
		update     kargoapi.GitRepoUpdate
		commits    []kargoapi.GitCommit
		assertions func(t *testing.T, readBranch string, commitIndex int, err error)
	}{
		{
			name: "update's RepoURL does not match any subscription",
			update: kargoapi.GitRepoUpdate{
				RepoURL:    "fake-url",
				ReadBranch: testBranch,
			},
			assertions: func(t *testing.T, readBranch string, commitIndex int, err error) {
				require.NoError(t, err)
				require.Equal(t, testBranch, readBranch)
				require.Equal(t, -1, commitIndex)
			},
		},
		{
			name: "subscription-loop avoided",
			update: kargoapi.GitRepoUpdate{
				RepoURL:     "fake-url",
				WriteBranch: testBranch,
			},
			commits: []kargoapi.GitCommit{
				{
					RepoURL: "fake-url",
					Branch:  testBranch,
				},
			},
			assertions: func(t *testing.T, _ string, _ int, err error) {
				require.ErrorContains(t, err, "because it will form a subscription loop")
			},
		},
		{
			name: "success",
			update: kargoapi.GitRepoUpdate{
				RepoURL: "fake-url",
			},
			commits: []kargoapi.GitCommit{
				{
					RepoURL: "fake-url",
					ID:      "fake-commit-id",
					Branch:  testBranch,
				},
			},
			assertions: func(t *testing.T, readBranch string, commitIndex int, err error) {
				require.NoError(t, err)
				require.Equal(t, "fake-commit-id", readBranch)
				require.Equal(t, 0, commitIndex)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			readBranch, commitIndex, err := getReadRef(testCase.update, testCase.commits)
			testCase.assertions(t, readBranch, commitIndex, err)
		})
	}
}

func TestGetRepoCredentials(t *testing.T) {
	testCases := []struct {
		name          string
		credentialsDB credentials.Database
		assertions    func(*testing.T, *git.RepoCredentials, error)
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
			assertions: func(t *testing.T, _ *git.RepoCredentials, err error) {
				require.ErrorContains(t, err, "error obtaining credentials")
				require.ErrorContains(t, err, "something went wrong")
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
			assertions: func(t *testing.T, creds *git.RepoCredentials, err error) {
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
			assertions: func(t *testing.T, creds *git.RepoCredentials, err error) {
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
			creds, err := getRepoCredentialsFn(testCase.credentialsDB)(
				context.Background(),
				"fake-namespace",
				"fake-repo-url",
			)
			testCase.assertions(t, creds, err)
		})
	}
}

func TestMoveRepoContents(t *testing.T) {
	const subdirCount = 50
	const fileCount = 50
	// Create dummy repo dir
	srcDir, err := createDummyRepoDir(t, subdirCount, fileCount)
	require.NoError(t, err)
	// Double-check the setup
	dirEntries, err := os.ReadDir(srcDir)
	require.NoError(t, err)
	require.Len(t, dirEntries, subdirCount+fileCount+1)
	// Create destination dir
	destDir := t.TempDir()
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
	dir, err := createDummyRepoDir(t, subdirCount, fileCount)
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

func TestApplyCopyPatches(t *testing.T) {
	testCases := []struct {
		name       string
		setup      func(*testing.T) (workingDir string, freightRepos map[string]string)
		update     kargoapi.GitRepoUpdate
		assertions func(t *testing.T, workingDir string, changes []string, err error)
	}{
		{
			name: "no patches",
			setup: func(*testing.T) (string, map[string]string) {
				return "", nil
			},
			update: kargoapi.GitRepoUpdate{},
			assertions: func(t *testing.T, _ string, changes []string, err error) {
				require.NoError(t, err)
				require.Empty(t, changes)
			},
		},
		{
			name: "no copy patches",
			setup: func(*testing.T) (string, map[string]string) {
				return "", nil
			},
			update: kargoapi.GitRepoUpdate{
				Patches: []kargoapi.PatchOperation{
					{}, {}, {},
				},
			},
			assertions: func(t *testing.T, _ string, changes []string, err error) {
				require.NoError(t, err)
				require.Empty(t, changes)
			},
		},
		{
			name: "no matching Freight repository URL",
			setup: func(*testing.T) (string, map[string]string) {
				return "", nil
			},
			update: kargoapi.GitRepoUpdate{
				Patches: []kargoapi.PatchOperation{
					{
						Copy: &kargoapi.CopyPatchOperation{
							RepoURL: "https://example.com/repo.git",
						},
					},
				},
			},
			assertions: func(t *testing.T, _ string, changes []string, err error) {
				require.ErrorContains(t, err, "no Freight repository found for URL")
				require.ErrorContains(t, err, "https://example.com/repo.git")
				require.Empty(t, changes)
			},
		},
		{
			name: "error copying",
			setup: func(t *testing.T) (string, map[string]string) {
				return t.TempDir(), nil
			},
			update: kargoapi.GitRepoUpdate{
				Patches: []kargoapi.PatchOperation{
					{
						Copy: &kargoapi.CopyPatchOperation{
							Source: "../../../../file.txt", // Traverses out of the working dir
						},
					},
				},
			},
			assertions: func(t *testing.T, _ string, changes []string, err error) {
				require.ErrorContains(t, err, "error performing copy operation")
				require.Empty(t, changes)
			},
		},
		{
			name: "success",
			setup: func(t *testing.T) (string, map[string]string) {
				workingDir := t.TempDir()
				require.NoError(
					t,
					os.WriteFile(filepath.Join(workingDir, "file.txt"), []byte("fake-content"), 0o600),
				)

				freightDir := t.TempDir()
				require.NoError(
					t,
					os.WriteFile(filepath.Join(freightDir, "file.txt"), []byte("fake-content"), 0o600),
				)

				return workingDir, map[string]string{
					libGit.NormalizeURL("https://example.com/repo.git"): freightDir,
				}
			},
			update: kargoapi.GitRepoUpdate{
				Patches: []kargoapi.PatchOperation{
					{
						Copy: &kargoapi.CopyPatchOperation{
							Source:      "file.txt",
							Destination: "copy/file.txt",
						},
					},
					{
						Copy: &kargoapi.CopyPatchOperation{
							RepoURL:     "https://example.com/repo.git",
							Source:      "file.txt",
							Destination: "copy/freight-file.txt",
						},
					},
				},
			},
			assertions: func(t *testing.T, workingDir string, changes []string, err error) {
				require.NoError(t, err)

				require.Len(t, changes, 2)
				require.Equal(t, "copied file.txt to copy/file.txt", changes[0])
				require.Equal(t, "copied file.txt from https://example.com/repo.git to copy/freight-file.txt", changes[1])

				// Check the working directory
				dirEntries, err := os.ReadDir(filepath.Join(workingDir, "copy"))
				require.NoError(t, err)
				require.Len(t, dirEntries, 2)
				require.Equal(t, "file.txt", dirEntries[0].Name())
				require.Equal(t, "freight-file.txt", dirEntries[1].Name())
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			workingDir, freightRepos := testCase.setup(t)
			changes, err := applyCopyPatches(workingDir, freightRepos, testCase.update)
			testCase.assertions(t, workingDir, changes, err)
		})
	}
}

func TestApplyCopyPatch(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func(*testing.T) (sourceDir string, targetDir string)
		source      string
		destination string
		assertions  func(t *testing.T, targetDir string, err error)
	}{
		{
			name: "source path outside of source directory",
			setup: func(t *testing.T) (string, string) {
				return t.TempDir(), ""
			},
			source: "../../../../file.txt",
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "not within the repository root")
			},
		},
		{
			name: "source path does not exist",
			setup: func(t *testing.T) (string, string) {
				return t.TempDir(), ""
			},
			source: "file.txt",
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "error getting info for source path")
				require.ErrorIs(t, err, os.ErrNotExist)
			},
		},
		{
			name: "source path is a directory",
			setup: func(t *testing.T) (string, string) {
				sourceDir := t.TempDir()
				require.NoError(t, os.Mkdir(filepath.Join(sourceDir, "dir"), 0o755))
				require.NoError(
					t,
					os.WriteFile(filepath.Join(sourceDir, "dir", "file.txt"), []byte("fake-content"), 0o600),
				)
				return sourceDir, t.TempDir()
			},
			source:      "dir",
			destination: "dir",
			assertions: func(t *testing.T, targetDir string, err error) {
				require.NoError(t, err)

				dirEntries, err := os.ReadDir(filepath.Join(targetDir, "dir"))
				require.NoError(t, err)
				require.Len(t, dirEntries, 1)
				require.Equal(t, "file.txt", dirEntries[0].Name())
			},
		},
		{
			name: "source path is a file",
			setup: func(t *testing.T) (string, string) {
				sourceDir := t.TempDir()
				require.NoError(
					t,
					os.WriteFile(filepath.Join(sourceDir, "file.txt"), []byte("fake-content"), 0o600),
				)
				return sourceDir, t.TempDir()
			},
			source:      "file.txt",
			destination: "file.txt",
			assertions: func(t *testing.T, targetDir string, err error) {
				require.NoError(t, err)

				dirEntries, err := os.ReadDir(targetDir)
				require.NoError(t, err)
				require.Len(t, dirEntries, 1)
				require.Equal(t, "file.txt", dirEntries[0].Name())
			},
		},
		{
			name: "source path is a symlink",
			setup: func(t *testing.T) (string, string) {
				sourceDir := t.TempDir()
				require.NoError(t, os.Symlink("anywhere", filepath.Join(sourceDir, "symlink")))
				return sourceDir, t.TempDir()
			},
			source: "symlink",
			assertions: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "unsupported file type for source path")
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			sourceDir, targetDir := testCase.setup(t)
			err := applyCopyPatch(sourceDir, targetDir, kargoapi.CopyPatchOperation{
				Source:      testCase.source,
				Destination: testCase.destination,
			})
			testCase.assertions(t, targetDir, err)
		})
	}
}

func TestBuildCommitMessage(t *testing.T) {
	testCases := []struct {
		name          string
		changeSummary []string
		assertions    func(t *testing.T, msg string)
	}{
		{
			// This shouldn't really happen, but we're careful to handle it anyway,
			// so we might as well test it.
			name:          "nil change summary",
			changeSummary: nil,
			assertions: func(t *testing.T, msg string) {
				require.Equal(t, "Kargo applied some changes", msg)
			},
		},
		{
			// This shouldn't really happen, but we're careful to handle it anyway,
			// so we might as well test it.
			name:          "empty change summary",
			changeSummary: []string{},
			assertions: func(t *testing.T, msg string) {
				require.Equal(t, "Kargo applied some changes", msg)
			},
		},
		{
			name: "change summary contains one item",
			changeSummary: []string{
				"fake-change",
			},
			assertions: func(t *testing.T, msg string) {
				require.Equal(t, "fake-change", msg)
			},
		},
		{
			name: "change summary contains multiple items",
			changeSummary: []string{
				"fake-change",
				"another-fake-change",
			},
			assertions: func(t *testing.T, msg string) {
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
			testCase.assertions(t, buildCommitMessage(testCase.changeSummary))
		})
	}
}

func createDummyRepoDir(t *testing.T, dirCount, fileCount int) (string, error) {
	t.Helper()
	// Create a temporary directory
	dir := t.TempDir()
	// Add a dummy .git/ subdir
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		return dir, err
	}
	// Add some dummy dirs
	for i := 0; i < dirCount; i++ {
		if err := os.Mkdir(
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
