package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSymlinks(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) (root string, dirPath string)
		maxDepth   int
		assertions func(*testing.T, error)
	}{
		{
			name: "no symlinks",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0o600))
				return root, root
			},
			maxDepth: 100,
			assertions: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "symlink within root",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0o600))
				require.NoError(t, os.Symlink(filepath.Join(root, "file.txt"), filepath.Join(root, "link.txt")))
				return root, root
			},
			maxDepth: 100,
			assertions: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "symlink outside root",
			setup: func(t *testing.T) (string, string) {
				dir := absoluteTempDir(t)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "outside.txt"), []byte("content"), 0o600))
				root := filepath.Join(dir, "root")
				require.NoError(t, os.Mkdir(root, 0o700))
				require.NoError(t, os.Symlink(filepath.Join(dir, "outside.txt"), filepath.Join(root, "link.txt")))
				return root, root
			},
			maxDepth: 100,
			assertions: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "symlink at link.txt points outside the path boundary")
			},
		},
		{
			name: "symlink to directory",
			setup: func(t *testing.T) (string, string) {
				dir := absoluteTempDir(t)
				subDir := filepath.Join(dir, "subdir")
				require.NoError(t, os.Mkdir(subDir, 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("content"), 0o600))
				require.NoError(t, os.Symlink(subDir, filepath.Join(dir, "link")))
				return dir, dir
			},
			maxDepth: 100,
			assertions: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "circular symlink",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				subDir := filepath.Join(root, "subdir")
				require.NoError(t, os.Mkdir(subDir, 0o700))
				require.NoError(t, os.Symlink(root, filepath.Join(subDir, "parent")))
				return root, root
			},
			maxDepth: 100,
			assertions: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "symlink with relative path",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0o600))
				require.NoError(t, os.Symlink("file.txt", filepath.Join(root, "link.txt")))
				return root, root
			},
			maxDepth: 100,
			assertions: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "symlink to non-existent file",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.Symlink(filepath.Join(root, "non-existent.txt"), filepath.Join(root, "link.txt")))
				return root, root
			},
			maxDepth: 100,
			assertions: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "failed to resolve symlink")
			},
		},
		{
			name: "recursion depth limit exceeded",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				subDir := root
				// Create a deep directory structure
				for i := 0; i < 10; i++ { // Exceeds depth limit of 5
					subDir = filepath.Join(subDir, fmt.Sprintf("level%d", i))
					require.NoError(t, os.Mkdir(subDir, 0o700))
				}
				return root, root
			},
			maxDepth: 5,
			assertions: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "maximum recursion depth exceeded")
			},
		},
		{
			name: "root path does not exist",
			setup: func(t *testing.T) (string, string) {
				root := filepath.Join(absoluteTempDir(t), "nonexistent")
				return root, root
			},
			maxDepth: 100,
			assertions: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "root path")
			},
		},
		{
			name: "unlimited depth with -1",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				subDir := root
				// Create a reasonably deep directory structure
				for i := 0; i < 15; i++ {
					subDir = filepath.Join(subDir, fmt.Sprintf("level%d", i))
					require.NoError(t, os.Mkdir(subDir, 0o700))
				}
				return root, root
			},
			maxDepth: -1, // Unlimited depth
			assertions: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "different root and dir paths",
			setup: func(t *testing.T) (string, string) {
				tempDir := absoluteTempDir(t)
				root := tempDir
				subDir := filepath.Join(tempDir, "subdir")
				require.NoError(t, os.Mkdir(subDir, 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("content"), 0o600))
				return root, subDir
			},
			maxDepth: 100,
			assertions: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "root path is not accessible",
			setup: func(t *testing.T) (string, string) {
				// Try to use a path that doesn't exist
				nonExistentRoot := filepath.Join(absoluteTempDir(t), "does-not-exist")
				return nonExistentRoot, nonExistentRoot
			},
			maxDepth: 100,
			assertions: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.ErrorContains(t, err, "root path")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, dirPath := tt.setup(t)
			err := ValidateSymlinks(root, dirPath, tt.maxDepth)
			tt.assertions(t, err)
		})
	}
}

func Test_validateSymlinks(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T) (root string, dirPath string)
		maxDepth     int
		setupVisited func(t *testing.T, root, dirPath string) map[string]struct{}
		assertions   func(*testing.T, map[string]struct{}, error)
	}{
		{
			name: "no symlinks",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0o600))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.NoError(t, err)
				assert.Len(t, visited, 1)
			},
		},
		{
			name: "symlink within root",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0o600))
				require.NoError(t, os.Symlink(filepath.Join(root, "file.txt"), filepath.Join(root, "link.txt")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.NoError(t, err)
				assert.Len(t, visited, 2)
			},
		},
		{
			name: "symlink outside root",
			setup: func(t *testing.T) (string, string) {
				dir := absoluteTempDir(t)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "outside.txt"), []byte("content"), 0o600))
				root := filepath.Join(dir, "root")
				require.NoError(t, os.Mkdir(root, 0o700))
				require.NoError(t, os.Symlink(filepath.Join(dir, "outside.txt"), filepath.Join(root, "link.txt")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.ErrorContains(t, err, "symlink at link.txt points outside the path boundary")
				assert.Len(t, visited, 1)
			},
		},
		{
			name: "symlink to directory",
			setup: func(t *testing.T) (string, string) {
				dir := absoluteTempDir(t)
				subDir := filepath.Join(dir, "subdir")
				require.NoError(t, os.Mkdir(subDir, 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("content"), 0o600))
				require.NoError(t, os.Symlink(subDir, filepath.Join(dir, "link")))
				return dir, dir
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.NoError(t, err)
				assert.Len(t, visited, 2)
			},
		},
		{
			name: "circular symlink",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				subDir := filepath.Join(root, "subdir")
				require.NoError(t, os.Mkdir(subDir, 0o700))
				require.NoError(t, os.Symlink(root, filepath.Join(subDir, "parent")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.NoError(t, err)
				assert.Len(t, visited, 2)
			},
		},
		{
			name: "symlink with relative path",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0o600))
				require.NoError(t, os.Symlink("file.txt", filepath.Join(root, "link.txt")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.NoError(t, err)
				assert.Len(t, visited, 2)
			},
		},
		{
			name: "symlink to non-existent file",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.Symlink(filepath.Join(root, "non-existent.txt"), filepath.Join(root, "link.txt")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.Error(t, err)
				assert.Len(t, visited, 1)
			},
		},
		{
			name: "symlink chain within root",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0o600))
				require.NoError(t, os.Symlink(filepath.Join(root, "file.txt"), filepath.Join(root, "link1.txt")))
				require.NoError(t, os.Symlink(filepath.Join(root, "link1.txt"), filepath.Join(root, "link2.txt")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.NoError(t, err)
				assert.Len(t, visited, 2)
			},
		},
		{
			name: "symlink to directory outside root",
			setup: func(t *testing.T) (string, string) {
				dir := absoluteTempDir(t)
				outsideDir := filepath.Join(dir, "outside")
				require.NoError(t, os.Mkdir(outsideDir, 0o700))
				root := filepath.Join(dir, "root")
				require.NoError(t, os.Mkdir(root, 0o700))
				require.NoError(t, os.Symlink(outsideDir, filepath.Join(root, "link")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.ErrorContains(t, err, "symlink at link points outside the path boundary")
				assert.Len(t, visited, 1)
			},
		},
		{
			name: "invalid symlink target",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.Symlink("non-existent.txt", filepath.Join(root, "invalidLink.txt")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.ErrorContains(t, err, "failed to resolve symlink")
				assert.Len(t, visited, 1)
			},
		},
		{
			name: "multiple links to same target",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0o600))
				require.NoError(t, os.Symlink(filepath.Join(root, "file.txt"), filepath.Join(root, "link1.txt")))
				require.NoError(t, os.Symlink(filepath.Join(root, "file.txt"), filepath.Join(root, "link2.txt")))
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.NoError(t, err)
				assert.Len(t, visited, 2)
			},
		},
		{
			name:     "recursion depth limit exceeded",
			maxDepth: 5,
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				subDir := root
				// Create a deep directory structure
				for i := 0; i < 10; i++ { // Exceeds depth limit of 5
					subDir = filepath.Join(subDir, fmt.Sprintf("level%d", i))
					require.NoError(t, os.Mkdir(subDir, 0o700))
				}
				return root, root
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.ErrorContains(t, err, "maximum recursion depth exceeded")
				assert.Len(t, visited, 6) // Root and 5 levels of subdirectories
			},
		},
		{
			name: "pre-visited directory should be skipped",
			setup: func(t *testing.T) (string, string) {
				root := absoluteTempDir(t)
				subDir := filepath.Join(root, "subdir")
				require.NoError(t, os.Mkdir(subDir, 0o700))
				require.NoError(t, os.WriteFile(filepath.Join(subDir, "file.txt"), []byte("content"), 0o600))
				return root, subDir
			},
			setupVisited: func(t *testing.T, _, dirPath string) map[string]struct{} {
				visited := make(map[string]struct{}, 1)
				absDir, err := filepath.Abs(dirPath)
				require.NoError(t, err)
				visited[absDir] = struct{}{}
				return visited
			},
			assertions: func(t *testing.T, visited map[string]struct{}, err error) {
				assert.NoError(t, err)
				// Should not add anything new since directory was already visited
				assert.Len(t, visited, 1)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, dirPath := tt.setup(t)

			var visited map[string]struct{}
			if tt.setupVisited != nil {
				visited = tt.setupVisited(t, root, dirPath)
			} else {
				visited = make(map[string]struct{})
			}

			maxDepth := tt.maxDepth
			if maxDepth <= 0 {
				maxDepth = 100
			}
			err := validateSymlinks(root, dirPath, visited, 0, maxDepth)
			tt.assertions(t, visited, err)
		})
	}
}

// absoluteTempDir returns the absolute path of a temporary directory created
// by t.TempDir(). This is useful when working with symlinks, as the temporary
// directory path may actually be a symlink on some platforms like macOS.
func absoluteTempDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	absDir, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	return absDir
}
