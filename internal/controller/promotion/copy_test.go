package promotion

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/internal/controller/git"
	libGit "github.com/akuity/kargo/internal/git"
)

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
			changes, err := applyCopyPatches(
				context.TODO(),
				testCase.update,
				kargoapi.FreightReference{},
				"", "", "",
				workingDir,
				freightRepos,
				git.RepoCredentials{},
			)
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
