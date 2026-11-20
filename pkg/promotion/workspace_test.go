package promotion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkspacePath(t *testing.T) {
	testCases := []struct {
		name   string
		path   string
		assert func(*testing.T, string, error)
	}{
		{
			name: "relative path",
			path: "a/b",
			assert: func(t *testing.T, rel string, err error) {
				require.NoError(t, err)
				require.Equal(t, "a/b", rel)
			},
		},
		{
			name: "path with leading dot",
			path: "./a",
			assert: func(t *testing.T, rel string, err error) {
				require.NoError(t, err)
				require.Equal(t, "a", rel)
			},
		},
		{
			name: "path that climbs within the workspace",
			path: "a/../b",
			assert: func(t *testing.T, rel string, err error) {
				require.NoError(t, err)
				require.Equal(t, "b", rel)
			},
		},
		{
			name: "absolute path is relative to the workspace",
			path: "/a",
			assert: func(t *testing.T, rel string, err error) {
				require.NoError(t, err)
				require.Equal(t, "a", rel)
			},
		},
		{
			name: "empty path is the workspace",
			path: "",
			assert: func(t *testing.T, rel string, err error) {
				require.NoError(t, err)
				require.Equal(t, ".", rel)
			},
		},
		{
			name: "dot is the workspace",
			path: ".",
			assert: func(t *testing.T, rel string, err error) {
				require.NoError(t, err)
				require.Equal(t, ".", rel)
			},
		},
		{
			name: "parent of the workspace",
			path: "..",
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, `path ".."`)
				require.ErrorContains(t, err, "working directory")
			},
		},
		{
			name: "path under the parent of the workspace",
			path: "../a",
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, `path "../a"`)
				require.ErrorContains(t, err, "working directory")
			},
		},
		{
			name: "path that climbs out after descending",
			path: "a/../../b",
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, `path "a/../../b"`)
				require.ErrorContains(t, err, "working directory")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			rel, err := WorkspacePath(testCase.path)
			testCase.assert(t, rel, err)
		})
	}
}

func TestExternalPath(t *testing.T) {
	workDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(workDir, "real"), 0o755))
	require.NoError(
		t,
		os.WriteFile(filepath.Join(workDir, "real", "x"), []byte("x"), 0o600),
	)
	require.NoError(t, os.Symlink("real", filepath.Join(workDir, "in")))
	require.NoError(t, os.Symlink("..", filepath.Join(workDir, "out")))
	require.NoError(
		t,
		os.Symlink("/nonexistent/abs", filepath.Join(workDir, "dangling-out")),
	)

	root, err := os.OpenRoot(workDir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })
	stepCtx := &StepContext{WorkDir: workDir, Root: root}

	testCases := []struct {
		name   string
		path   string
		assert func(*testing.T, string, error)
	}{
		{
			name: "existing file",
			path: "real/x",
			assert: func(t *testing.T, abs string, err error) {
				require.NoError(t, err)
				require.Equal(t, filepath.Join(workDir, "real", "x"), abs)
			},
		},
		{
			name: "file through a symlink inside the workspace",
			path: "in/x",
			assert: func(t *testing.T, abs string, err error) {
				require.NoError(t, err)
				require.Equal(t, filepath.Join(workDir, "in", "x"), abs)
			},
		},
		{
			name: "file through a symlink that escapes the workspace",
			path: "out/x",
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, `path "out/x"`)
			},
		},
		{
			name: "dangling symlink to an absolute path",
			path: "dangling-out",
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, `path "dangling-out"`)
			},
		},
		{
			name: "missing file under an existing directory",
			path: "real/new/x",
			assert: func(t *testing.T, abs string, err error) {
				require.NoError(t, err)
				require.Equal(t, filepath.Join(workDir, "real", "new", "x"), abs)
			},
		},
		{
			name: "missing file under an escaping symlink",
			path: "out/new/x",
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, `path "out/new/x"`)
			},
		},
		{
			name: "the workspace itself",
			path: "",
			assert: func(t *testing.T, abs string, err error) {
				require.NoError(t, err)
				require.Equal(t, workDir, abs)
			},
		},
		{
			name: "path that climbs out of the workspace",
			path: "../x",
			assert: func(t *testing.T, _ string, err error) {
				require.ErrorContains(t, err, `path "../x"`)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			abs, err := ExternalPath(stepCtx, testCase.path)
			testCase.assert(t, abs, err)
		})
	}
}
