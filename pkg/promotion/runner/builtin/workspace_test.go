package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_createTempFile(t *testing.T) {
	workDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(workDir, "dir"), 0o755))
	root := newTestRoot(t, workDir)

	f, name, err := createTempFile(root, "dir", "file.txt.tmp")
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	require.Equal(t, "dir", filepath.Dir(name))
	require.True(t, strings.HasPrefix(filepath.Base(name), "file.txt.tmp"))
	info, err := os.Stat(filepath.Join(workDir, name))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	_, starred, err := createTempFile(root, "dir", "pre-*.tmp")
	require.NoError(t, err)
	require.Regexp(t, `^dir/pre-[A-Z2-7]{26}\.tmp$`, starred)

	_, _, err = createTempFile(root, "missing", "x")
	require.ErrorIs(t, err, os.ErrNotExist)
}

func Test_mkdirTemp(t *testing.T) {
	workDir := t.TempDir()
	root := newTestRoot(t, workDir)
	name, err := mkdirTemp(root, ".", ".untar-*")
	require.NoError(t, err)
	require.Regexp(t, `^\.untar-[A-Z2-7]{26}$`, name)
	require.DirExists(t, filepath.Join(workDir, name))
}

func Test_replace(t *testing.T) {
	t.Run("destination does not exist", func(t *testing.T) {
		workDir := t.TempDir()
		root := newTestRoot(t, workDir)
		require.NoError(t, root.WriteFile("src", []byte("x"), 0o600))
		require.NoError(t, replace(root, "src", "dst"))
		require.NoFileExists(t, filepath.Join(workDir, "src"))
		b, err := os.ReadFile(filepath.Join(workDir, "dst"))
		require.NoError(t, err)
		require.Equal(t, "x", string(b))
	})
	t.Run("destination is a non-empty directory", func(t *testing.T) {
		workDir := t.TempDir()
		root := newTestRoot(t, workDir)
		require.NoError(t, root.MkdirAll("src", 0o755))
		require.NoError(t, root.WriteFile("src/new", []byte("x"), 0o600))
		require.NoError(t, root.MkdirAll("dst", 0o755))
		require.NoError(t, root.WriteFile("dst/old", []byte("x"), 0o600))
		require.NoError(t, replace(root, "src", "dst"))
		require.NoDirExists(t, filepath.Join(workDir, "src"))
		require.FileExists(t, filepath.Join(workDir, "dst", "new"))
		require.NoFileExists(t, filepath.Join(workDir, "dst", "old"))
	})
	t.Run("destination is outside the workspace", func(t *testing.T) {
		workDir, outside := escapingWorkspace(t)
		root := newTestRoot(t, workDir)
		require.NoError(t, root.WriteFile("src", []byte("x"), 0o600))
		require.ErrorContains(t, replace(root, "src", "out/dst"), "path escapes from parent")
		require.NoFileExists(t, filepath.Join(outside, "dst"))
	})
}

func Test_resolvePath(t *testing.T) {
	workDir, outside := escapingWorkspace(t)
	require.NoError(t, os.Mkdir(filepath.Join(workDir, "real"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "real", "x"), []byte("x"), 0o600))
	require.NoError(t, os.Symlink("real", filepath.Join(workDir, "in")))
	require.NoError(t, os.Symlink("in", filepath.Join(workDir, "chain")))
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret"), []byte("x"), 0o600))
	root := newTestRoot(t, workDir)

	testCases := []struct {
		name   string
		path   string
		assert func(*testing.T, string, error)
	}{
		{
			name: "not a symlink",
			path: "real/x",
			assert: func(t *testing.T, resolved string, err error) {
				require.NoError(t, err)
				require.Equal(t, "real/x", resolved)
			},
		},
		{
			name: "symlink inside the workspace",
			path: "in",
			assert: func(t *testing.T, resolved string, err error) {
				require.NoError(t, err)
				require.Equal(t, "real", resolved)
			},
		},
		{
			name: "chain of symlinks inside the workspace",
			path: "chain",
			assert: func(t *testing.T, resolved string, err error) {
				require.NoError(t, err)
				require.Equal(t, "real", resolved)
			},
		},
		{
			name: "path through a symlink inside the workspace",
			path: "in/x",
			assert: func(t *testing.T, resolved string, err error) {
				require.NoError(t, err)
				require.Equal(t, "real/x", resolved)
			},
		},
		{
			name: "symlink outside the workspace",
			path: "out",
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "outside the working directory")
			},
		},
		{
			name: "path through a symlink outside the workspace",
			path: "out/secret",
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, "outside the working directory")
			},
		},
		{
			name: "missing path is returned unchanged",
			path: "real/missing",
			assert: func(t *testing.T, resolved string, err error) {
				require.NoError(t, err)
				require.Equal(t, "real/missing", resolved)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			resolved, err := resolvePath(root, testCase.path)
			testCase.assert(t, resolved, err)
		})
	}
}

func Test_statTree_danglingLinkDoesNotHideEscapingLink(t *testing.T) {
	workDir, _ := escapingWorkspace(t)
	root := newTestRoot(t, workDir)
	require.NoError(t, root.Mkdir("dir", 0o755))
	require.NoError(t, root.Symlink("missing", "dir/a-dangling"))
	require.NoError(t, root.Symlink("../out", "dir/z-escaping"))
	err := statTree(root, "dir")
	require.Error(t, err)
	require.NotErrorIs(t, err, os.ErrNotExist)
	require.ErrorIs(t, statTree(root, "absent"), os.ErrNotExist)
}

func Test_replace_symlinkedDestination(t *testing.T) {
	t.Run("destination is a symlink to a directory", func(t *testing.T) {
		workDir := t.TempDir()
		root := newTestRoot(t, workDir)
		require.NoError(t, root.Mkdir("real", 0o755))
		require.NoError(t, root.WriteFile("real/old", []byte("old"), 0o600))
		require.NoError(t, root.Symlink("real", "link"))
		require.NoError(t, root.Mkdir("src", 0o755))
		require.NoError(t, root.WriteFile("src/new", []byte("new"), 0o600))
		require.NoError(t, replace(root, "src", "link"))
		require.FileExists(t, filepath.Join(workDir, "real", "new"))
		require.NoFileExists(t, filepath.Join(workDir, "real", "old"))
		fi, err := os.Lstat(filepath.Join(workDir, "link"))
		require.NoError(t, err)
		require.NotZero(t, fi.Mode()&os.ModeSymlink)
	})
	t.Run("destination is a symlink to a file", func(t *testing.T) {
		workDir := t.TempDir()
		root := newTestRoot(t, workDir)
		require.NoError(t, root.WriteFile("target.txt", []byte("old"), 0o600))
		require.NoError(t, root.Symlink("target.txt", "link.txt"))
		require.NoError(t, root.WriteFile("src.txt", []byte("new"), 0o600))
		require.NoError(t, replace(root, "src.txt", "link.txt"))
		b, err := os.ReadFile(filepath.Join(workDir, "target.txt"))
		require.NoError(t, err)
		require.Equal(t, "new", string(b))
		fi, err := os.Lstat(filepath.Join(workDir, "link.txt"))
		require.NoError(t, err)
		require.NotZero(t, fi.Mode()&os.ModeSymlink)
	})
}
