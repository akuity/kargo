package fs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCopyDir(t *testing.T) {
	tests := []struct {
		name       string
		nodes      []testDirNode
		setup      func(t *testing.T) (src, dst string)
		assertions func(t *testing.T, dst string, err error)
	}{
		{
			name: "empty directory",
			assertions: func(t *testing.T, dst string, err error) {
				require.NoError(t, err)
				assertDirEqual(t, dst, nil)
			},
		},
		{
			name: "directory with files and subdirectories",
			nodes: []testDirNode{
				{Path: "dir1", IsDir: true, Perm: 0o755},
				{Path: "dir1/file1.txt", Data: []byte("file1 contents"), Perm: 0o600},
				{Path: "dir1/subdir", IsDir: true, Perm: 0o755},
				{Path: "dir1/subdir/file2.txt", Data: []byte("file2 contents"), Perm: 0o600},
				{Path: "file3.txt", Data: []byte("file3 contents"), Perm: 0o600},
			},
			assertions: func(t *testing.T, dst string, err error) {
				require.NoError(t, err)
				assertDirEqual(t, dst, []testDirNode{
					{Path: "dir1", IsDir: true, Perm: 0o755},
					{Path: "dir1/file1.txt", Data: []byte("file1 contents"), Perm: 0o600},
					{Path: "dir1/subdir", IsDir: true, Perm: 0o755},
					{Path: "dir1/subdir/file2.txt", Data: []byte("file2 contents"), Perm: 0o600},
					{Path: "file3.txt", Data: []byte("file3 contents"), Perm: 0o600},
				})
			},
		},
		{
			name: "destination is subdirectory of source",
			setup: func(t *testing.T) (src, dst string) {
				src = t.TempDir()
				dst = filepath.Join(src, "dir1")
				return src, dst
			},
			nodes: []testDirNode{
				{Path: "file1.txt", IsDir: true, Perm: 0o600},
				{Path: "dir1/subdir", IsDir: true, Perm: 0o755},
				{Path: "dir1/subdir/file2.txt", Data: []byte("file2 contents"), Perm: 0o600},
			},
			assertions: func(t *testing.T, dst string, err error) {
				require.NoError(t, err)
				assertDirEqual(t, dst, []testDirNode{
					{Path: "file1.txt", IsDir: true, Perm: 0o600},
					{Path: "subdir", IsDir: true, Perm: 0o755},
					{Path: "subdir/file2.txt", Data: []byte("file2 contents"), Perm: 0o600},
				})
			},
		},
		{
			name: "copy with permissions",
			nodes: []testDirNode{
				{Path: "dir1", IsDir: true, Perm: 0o700},
				{Path: "dir1/file1.txt", Data: []byte("file1 contents"), Perm: 0o600},
				{Path: "dir1/file2.txt", Data: []byte("file2 contents"), Perm: 0o600},
			},
			assertions: func(t *testing.T, dst string, err error) {
				require.NoError(t, err)
				assertDirEqual(t, dst, []testDirNode{
					{Path: "dir1", IsDir: true, Perm: 0o700},
					{Path: "dir1/file1.txt", Data: []byte("file1 contents"), Perm: 0o600},
					{Path: "dir1/file2.txt", Data: []byte("file2 contents"), Perm: 0o600},
				})
			},
		},
		{
			name: "ignores symlinks",
			nodes: []testDirNode{
				{Path: "dir1", IsDir: true, Perm: 0o755},
				{Path: "dir1/file1.txt", Data: []byte("file1 contents"), Perm: 0o600},
				{Path: "dir1/link1", Target: "file1.txt"},
			},
			assertions: func(t *testing.T, dst string, err error) {
				require.NoError(t, err)
				assertDirEqual(t, dst, []testDirNode{
					{Path: "dir1", IsDir: true, Perm: 0o755},
					{Path: "dir1/file1.txt", Data: []byte("file1 contents"), Perm: 0o600},
				})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup := tt.setup
			if setup == nil {
				setup = func(t *testing.T) (src, dst string) {
					src = t.TempDir()
					dst = t.TempDir()
					return src, dst
				}
			}
			src, dst := setup(t)
			if len(tt.nodes) > 0 {
				createTestDir(t, src, tt.nodes)
			}
			err := CopyDir(src, dst)
			tt.assertions(t, dst, err)
		})
	}
}

func TestCopyFile(t *testing.T) {
	const (
		srcData      = "Quick brown fox jumps over the lazy dog"
		existingData = "Lazy dog jumps over the quick brown fox"
	)

	tests := []struct {
		name       string
		setup      func(t *testing.T) (src, dst string)
		assertions func(t *testing.T, dst string, err error)
	}{
		{
			name: "copy regular file",
			setup: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()

				srcFilePath := filepath.Join(tempDir, "src")
				dstFilePath := filepath.Join(tempDir, "dst")

				require.NoError(t, os.WriteFile(srcFilePath, []byte(srcData), 0o600))

				return srcFilePath, dstFilePath
			},
			assertions: func(t *testing.T, dst string, err error) {
				require.NoError(t, err)

				dstData, err := os.ReadFile(dst)
				require.NoError(t, err)
				require.Equal(t, srcData, string(dstData))
			},
		},
		{
			name: "overwrite existing file",
			setup: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()

				srcFilePath := filepath.Join(tempDir, "src")
				dstFilePath := filepath.Join(tempDir, "dst")

				require.NoError(t, os.WriteFile(srcFilePath, []byte(srcData), 0o600))
				require.NoError(t, os.WriteFile(dstFilePath, []byte(existingData), 0o600))

				return srcFilePath, dstFilePath
			},
			assertions: func(t *testing.T, dst string, err error) {
				require.NoError(t, err)

				dstData, err := os.ReadFile(dst)
				require.NoError(t, err)
				require.Equal(t, srcData, string(dstData))
			},
		},
		{
			name: "copy with permissions",
			setup: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()

				srcFilePath := filepath.Join(tempDir, "src")
				dstFilePath := filepath.Join(tempDir, "dst")

				require.NoError(t, os.WriteFile(srcFilePath, []byte(srcData), 0o400))

				return srcFilePath, dstFilePath
			},
			assertions: func(t *testing.T, dst string, err error) {
				require.NoError(t, err)

				info, err := os.Stat(dst)
				require.NoError(t, err)
				require.Equal(t, os.FileMode(0o400), info.Mode())
			},
		},
		{
			name: "copy symbolic link",
			setup: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()

				targetFilePath := filepath.Join(tempDir, "target")
				linkFilePath := filepath.Join(tempDir, "link")
				dstFilePath := filepath.Join(tempDir, "dst")

				require.NoError(t, os.WriteFile(targetFilePath, []byte(srcData), 0o600))
				require.NoError(t, os.Symlink(targetFilePath, linkFilePath))

				return linkFilePath, dstFilePath
			},
			assertions: func(t *testing.T, dst string, err error) {
				require.NoError(t, err)

				dstData, err := os.ReadFile(dst)
				require.NoError(t, err)
				require.Equal(t, srcData, string(dstData))
			},
		},
		{
			name: "source file does not exist",
			setup: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()
				dstFilePath := filepath.Join(tempDir, "dst")
				return "non-existent-file", dstFilePath
			},
			assertions: func(t *testing.T, _ string, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "invalid destination path",
			setup: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()

				srcFilePath := filepath.Join(tempDir, "src")
				nonExistentPath := filepath.Join(tempDir, "non-existent-dir", "dst")

				require.NoError(t, os.WriteFile(srcFilePath, []byte(srcData), 0o600))

				return srcFilePath, nonExistentPath
			},
			assertions: func(t *testing.T, _ string, err error) {
				require.Error(t, err)
				require.ErrorContains(t, err, "no such file or directory")
			},
		},
		{
			name: "source is a directory",
			setup: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()

				srcDirPath := filepath.Join(tempDir, "src")
				dstFilePath := filepath.Join(tempDir, "dst")

				require.NoError(t, os.Mkdir(srcDirPath, 0o755))

				return srcDirPath, dstFilePath
			},
			assertions: func(t *testing.T, _ string, err error) {
				require.Error(t, err)
			},
		},
		{
			name: "destination is a directory",
			setup: func(t *testing.T) (string, string) {
				tempDir := t.TempDir()

				srcFilePath := filepath.Join(tempDir, "src")
				dstDirPath := filepath.Join(tempDir, "dst")

				require.NoError(t, os.WriteFile(srcFilePath, []byte(srcData), 0o600))
				require.NoError(t, os.Mkdir(dstDirPath, 0o755))

				return srcFilePath, dstDirPath
			},
			assertions: func(t *testing.T, _ string, err error) {
				require.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst := tt.setup(t)
			err := CopyFile(src, dst)
			tt.assertions(t, dst, err)
		})
	}
}

type testDirNode struct {
	Path   string
	Target string
	IsDir  bool
	Data   []byte
	Perm   os.FileMode
}

func createTestDir(t *testing.T, baseDir string, nodes []testDirNode) {
	for _, node := range nodes {
		path := filepath.Join(baseDir, node.Path)
		switch {
		case node.IsDir:
			require.NoError(t, os.MkdirAll(path, node.Perm))
		case node.Target != "":
			require.NoError(t, os.Symlink(node.Target, path))
		default:
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			require.NoError(t, os.WriteFile(path, node.Data, node.Perm))
		}
	}
}

func assertDirEqual(t *testing.T, dir string, expected []testDirNode) {
	var actual []testDirNode
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == dir {
			return nil
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		node := testDirNode{
			Path:  relPath,
			IsDir: d.IsDir(),
			Perm:  info.Mode().Perm(),
		}

		if d.Type().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			node.Data = data
		}

		if d.Type()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			node.Target = target
		}

		actual = append(actual, node)
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, expected, actual)
}
