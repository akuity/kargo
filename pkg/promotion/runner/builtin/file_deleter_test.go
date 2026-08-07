package builtin

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_fileDeleter_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name:   "path and paths not specified",
			config: promotion.Config{},
			expectedProblems: []string{
				"(root): Must validate one and only one schema",
			},
		},
		{
			name: "path is empty string",
			config: promotion.Config{
				"path": "",
			},
			expectedProblems: []string{
				"path: String length must be greater than or equal to 1",
			},
		},
		{
			name: "strict is not specified",
			config: promotion.Config{
				"path": "/path/to/delete",
			},
			// No expected problems because strict is optional with default: false
			expectedProblems: nil,
		},
		{
			name: "valid minimal config",
			config: promotion.Config{
				"path": "/path/to/delete",
			},
			expectedProblems: nil,
		},
		{
			name: "valid config with strict=false",
			config: promotion.Config{
				"path":   "/path/to/delete",
				"strict": false,
			},
			expectedProblems: nil,
		},
		{
			name: "valid config with strict=true",
			config: promotion.Config{
				"path":   "/path/to/delete",
				"strict": true,
			},
			expectedProblems: nil,
		},
		{
			name: "valid kitchen sink",
			config: promotion.Config{
				"path":   "/path/to/file/or/directory/to/delete",
				"strict": true,
			},
			expectedProblems: nil,
		},
		{
			name: "paths with multiple entries",
			config: promotion.Config{
				"paths": []any{"/path/to/a", "/path/to/b", "/path/to/c"},
			},
			expectedProblems: nil,
		},
		{
			name: "paths present but empty array",
			config: promotion.Config{
				"paths": []any{},
			},
			expectedProblems: []string{
				"paths: Array must have at least 1 items",
			},
		},
		{
			name: "paths containing an empty string",
			config: promotion.Config{
				"paths": []any{""},
			},
			expectedProblems: []string{
				"paths.0: String length must be greater than or equal to 1",
			},
		},
		{
			name: "pathsAreGlobs true with path",
			config: promotion.Config{
				"path":          "*.txt",
				"pathsAreGlobs": true,
			},
			expectedProblems: nil,
		},
		{
			name: "pathsAreGlobs true with paths",
			config: promotion.Config{
				"paths":         []any{"*.txt", "**/*.tmp"},
				"pathsAreGlobs": true,
			},
			expectedProblems: nil,
		},
		{
			name: "pathsAreGlobs with wrong type",
			config: promotion.Config{
				"path":          "/path/to/delete",
				"pathsAreGlobs": "true",
			},
			expectedProblems: []string{
				"pathsAreGlobs: Invalid type. Expected: boolean, given: string",
			},
		},
		{
			name: "unknown additional property",
			config: promotion.Config{
				"path":  "/path/to/delete",
				"bogus": "value",
			},
			expectedProblems: []string{
				"Additional property bogus is not allowed",
			},
		},
		{
			name: "valid glob kitchen sink",
			config: promotion.Config{
				"paths":         []any{"**/*.log", "build/"},
				"pathsAreGlobs": true,
				"strict":        true,
			},
			expectedProblems: nil,
		},
	}

	r := newFileDeleter(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*fileDeleter)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_fileDeleter_run(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(*testing.T) string
		cfg        builtin.DeleteConfig
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "succeeds deleting file",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				path := filepath.Join(tmpDir, "input.txt")
				require.NoError(t, os.WriteFile(path, []byte("test content"), 0o600))

				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Path: "input.txt",
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				_, statError := os.Stat("input.txt")
				assert.True(t, os.IsNotExist(statError))
			},
		},
		{
			name: "succeeds deleting directory",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				dirPath := filepath.Join(tmpDir, "dirToDelete")
				require.NoError(t, os.Mkdir(dirPath, 0o700))
				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Path: "dirToDelete",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				_, statErr := os.Stat(filepath.Join(workDir, "dirToDelete"))
				assert.True(t, os.IsNotExist(statErr))
			},
		},
		{
			name: "fails for non-existent path when strict is true",
			setupFiles: func(t *testing.T) string {
				return t.TempDir()
			},
			cfg: builtin.DeleteConfig{
				Path:   "nonExistentFile.txt",
				Strict: true,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
			},
		},
		{
			name: "succeeds for non-existent path when strict is false",
			setupFiles: func(t *testing.T) string {
				return t.TempDir()
			},
			cfg: builtin.DeleteConfig{
				Path:   "nonExistentFile.txt",
				Strict: false,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)
			},
		},
		{
			name: "removes symlink only",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				inDir := filepath.Join(tmpDir, "input")
				require.NoError(t, os.Mkdir(inDir, 0o755))

				filePath := filepath.Join(inDir, "input.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("test content"), 0o600))

				symlinkPath := filepath.Join(inDir, "symlink.txt")
				require.NoError(t, os.Symlink("input.txt", symlinkPath))

				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Path: "input/symlink.txt",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				require.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				_, statErr := os.Stat(filepath.Join(workDir, "input", "input.txt"))
				assert.NoError(t, statErr)

				_, statErr = os.Lstat(filepath.Join(workDir, "input", "symlink.txt"))
				assert.Error(t, statErr)
				assert.True(t, os.IsNotExist(statErr))
			},
		},
		{
			name: "removes a file within a symlink",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				inDir := filepath.Join(tmpDir, "bar")
				require.NoError(t, os.Mkdir(inDir, 0o755))

				filePath := filepath.Join(inDir, "file.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("test content"), 0o600))

				symlinkPath := filepath.Join(tmpDir, "foo")
				require.NoError(t, os.Symlink(inDir, symlinkPath))

				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Path: "foo/",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				require.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				_, statErr := os.Stat(filepath.Join(workDir, "foo", "file.txt"))
				assert.Error(t, statErr)
				assert.True(t, os.IsNotExist(statErr))

				_, statErr = os.Stat(filepath.Join(workDir, "bar", "file.txt"))
				assert.NoError(t, statErr)
			},
		},
		{
			name: "literal paths deletes multiple entries",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "a.txt"), []byte("a"), 0o600))
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "b.txt"), []byte("b"), 0o600))
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "c.md"), []byte("c"), 0o600))
				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Paths: []string{"a.txt", "b.txt"},
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				_, statErr := os.Stat(filepath.Join(workDir, "a.txt"))
				assert.True(t, os.IsNotExist(statErr))
				_, statErr = os.Stat(filepath.Join(workDir, "b.txt"))
				assert.True(t, os.IsNotExist(statErr))
				// Non-listed file is retained.
				_, statErr = os.Stat(filepath.Join(workDir, "c.md"))
				assert.NoError(t, statErr)
			},
		},
		{
			name: "literal paths with a parent directory and a child both listed",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.Mkdir(filepath.Join(tmpDir, "dir"), 0o700))
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "dir", "file.txt"), []byte("x"), 0o600))
				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Paths:  []string{"dir", "dir/file.txt"},
				Strict: true,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)
				_, statErr := os.Stat(filepath.Join(workDir, "dir"))
				assert.True(t, os.IsNotExist(statErr))
			},
		},
		{
			name: "literal paths with one missing entry succeeds when strict is false",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "a.txt"), []byte("a"), 0o600))
				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Paths:  []string{"a.txt", "missing.txt"},
				Strict: false,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)
				_, statErr := os.Stat(filepath.Join(workDir, "a.txt"))
				assert.True(t, os.IsNotExist(statErr))
			},
		},
		{
			name: "literal paths with one missing entry errors when strict is true",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "a.txt"), []byte("a"), 0o600))
				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Paths:  []string{"a.txt", "missing.txt"},
				Strict: true,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				// Nothing is deleted because strict mode fails on the missing
				// entry before any deletion takes place.
				_, statErr := os.Stat(filepath.Join(workDir, "a.txt"))
				assert.NoError(t, statErr)
			},
		},
		{
			name: "literal path with glob metacharacters is treated literally",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "a[1].txt"), []byte("lit"), 0o600))
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "a2.txt"), []byte("glob"), 0o600))
				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Path: "a[1].txt",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				_, statErr := os.Stat(filepath.Join(workDir, "a[1].txt"))
				assert.True(t, os.IsNotExist(statErr))
				_, statErr = os.Stat(filepath.Join(workDir, "a2.txt"))
				assert.NoError(t, statErr)
			},
		},
		{
			name: "glob matches multiple files and retains non-matching",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "a.txt"), []byte("a"), 0o600))
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "b.txt"), []byte("b"), 0o600))
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "c.md"), []byte("c"), 0o600))
				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Path:          "*.txt",
				PathsAreGlobs: true,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				_, statErr := os.Stat(filepath.Join(workDir, "a.txt"))
				assert.True(t, os.IsNotExist(statErr))
				_, statErr = os.Stat(filepath.Join(workDir, "b.txt"))
				assert.True(t, os.IsNotExist(statErr))
				_, statErr = os.Stat(filepath.Join(workDir, "c.md"))
				assert.NoError(t, statErr)
			},
		},
		{
			name: "glob with recursive ** matches nested files",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "sub", "deep"), 0o755))
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "a.tmp"), []byte("a"), 0o600))
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "sub", "b.tmp"), []byte("b"), 0o600))
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "sub", "deep", "c.tmp"), []byte("c"), 0o600))
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "sub", "deep", "keep.md"), []byte("k"), 0o600))
				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Path:          "**/*.tmp",
				PathsAreGlobs: true,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				_, statErr := os.Stat(filepath.Join(workDir, "a.tmp"))
				assert.True(t, os.IsNotExist(statErr))
				_, statErr = os.Stat(filepath.Join(workDir, "sub", "b.tmp"))
				assert.True(t, os.IsNotExist(statErr))
				_, statErr = os.Stat(filepath.Join(workDir, "sub", "deep", "c.tmp"))
				assert.True(t, os.IsNotExist(statErr))
				_, statErr = os.Stat(filepath.Join(workDir, "sub", "deep", "keep.md"))
				assert.NoError(t, statErr)
			},
		},
		{
			name: "glob patterns with a leading ./ are matched relative to workDir",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "out", "build"), 0o755))
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "out", "a.tmp"), []byte("a"), 0o600))
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "out", "build", "b.tmp"), []byte("b"), 0o600))
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "out", "keep.md"), []byte("k"), 0o600))
				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Paths:         []string{"./out/**/*.tmp", "./out/build/"},
				PathsAreGlobs: true,
				Strict:        true,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				_, statErr := os.Stat(filepath.Join(workDir, "out", "a.tmp"))
				assert.True(t, os.IsNotExist(statErr))
				_, statErr = os.Stat(filepath.Join(workDir, "out", "build"))
				assert.True(t, os.IsNotExist(statErr))
				// Non-matching file is retained.
				_, statErr = os.Stat(filepath.Join(workDir, "out", "keep.md"))
				assert.NoError(t, statErr)
			},
		},
		{
			name: "glob matching a directory deletes the whole directory",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "build", "out"), 0o755))
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "build", "out", "artifact"), []byte("x"), 0o600))
				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Path:          "buil*",
				PathsAreGlobs: true,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)
				_, statErr := os.Stat(filepath.Join(workDir, "build"))
				assert.True(t, os.IsNotExist(statErr))
			},
		},
		{
			name: "glob deletes a matched symlink as a link",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				filePath := filepath.Join(tmpDir, "target.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("test content"), 0o600))

				symlinkPath := filepath.Join(tmpDir, "link.txt")
				require.NoError(t, os.Symlink("target.txt", symlinkPath))

				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Path:          "link.*",
				PathsAreGlobs: true,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				// The symlink is removed, but its target is left intact.
				_, statErr := os.Lstat(filepath.Join(workDir, "link.txt"))
				assert.True(t, os.IsNotExist(statErr))
				_, statErr = os.Stat(filepath.Join(workDir, "target.txt"))
				assert.NoError(t, statErr)
			},
		},
		{
			name: "glob paths with multiple patterns deletes the union",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "a.log"), []byte("a"), 0o600))
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "b.tmp"), []byte("b"), 0o600))
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "c.md"), []byte("c"), 0o600))
				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Paths:         []string{"*.log", "*.tmp"},
				PathsAreGlobs: true,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				_, statErr := os.Stat(filepath.Join(workDir, "a.log"))
				assert.True(t, os.IsNotExist(statErr))
				_, statErr = os.Stat(filepath.Join(workDir, "b.tmp"))
				assert.True(t, os.IsNotExist(statErr))
				_, statErr = os.Stat(filepath.Join(workDir, "c.md"))
				assert.NoError(t, statErr)
			},
		},
		{
			name: "glob matches nothing and succeeds when strict is false",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "a.md"), []byte("a"), 0o600))
				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Path:          "*.txt",
				PathsAreGlobs: true,
				Strict:        false,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)
				// Nothing matched, so nothing was deleted.
				_, statErr := os.Stat(filepath.Join(workDir, "a.md"))
				assert.NoError(t, statErr)
			},
		},
		{
			name: "glob matches nothing and errors when strict is true",
			setupFiles: func(t *testing.T) string {
				return t.TempDir()
			},
			cfg: builtin.DeleteConfig{
				Path:          "*.txt",
				PathsAreGlobs: true,
				Strict:        true,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.ErrorContains(t, err, "no paths matched pattern")
				assert.ErrorContains(t, err, "*.txt")
			},
		},
		{
			name: "glob paths mixed match and no-match errors on the empty one when strict is true",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				require.NoError(t, os.WriteFile(
					filepath.Join(tmpDir, "a.log"), []byte("a"), 0o600))
				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Paths:         []string{"*.log", "*.txt"},
				PathsAreGlobs: true,
				Strict:        true,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)

				_, statErr := os.Stat(filepath.Join(workDir, "a.log"))
				assert.NoError(t, statErr)
			},
		},
		{
			name: "invalid glob pattern is a terminal error",
			setupFiles: func(t *testing.T) string {
				return t.TempDir()
			},
			cfg: builtin.DeleteConfig{
				Path:          "[",
				PathsAreGlobs: true,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				var termErr *promotion.TerminalError
				assert.True(t, errors.As(err, &termErr))
			},
		},
		{
			name: "literal path with a missing parent directory succeeds when strict is false",
			setupFiles: func(t *testing.T) string {
				return t.TempDir()
			},
			cfg: builtin.DeleteConfig{
				Path:   "no_such_dir/child.txt",
				Strict: false,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)
			},
		},
		{
			name: "literal path with a missing parent directory errors when strict is true",
			setupFiles: func(t *testing.T) string {
				return t.TempDir()
			},
			cfg: builtin.DeleteConfig{
				Path:   "no_such_dir/child.txt",
				Strict: true,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.ErrorContains(t, err, "does not exist")
			},
		},
		{
			name: "literal path attempting to traverse outside workDir errors",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				// A file that lives one level above the workDir.
				escapePath := filepath.Join(filepath.Dir(tmpDir), "escape-target")
				require.NoError(t, os.WriteFile(escapePath, []byte("x"), 0o600))
				t.Cleanup(func() { _ = os.Remove(escapePath) })
				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Path: "../escape-target",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				// The external target is not deleted.
				_, statErr := os.Stat(filepath.Join(filepath.Dir(workDir), "escape-target"))
				assert.NoError(t, statErr)
			},
		},
		{
			name: "literal path does not delete external files through a symlink",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				extDir := filepath.Join(filepath.Dir(tmpDir), "kargo-ext-literal")
				require.NoError(t, os.Mkdir(extDir, 0o755))
				require.NoError(t, os.WriteFile(
					filepath.Join(extDir, "secret.txt"), []byte("x"), 0o600))
				t.Cleanup(func() { _ = os.RemoveAll(extDir) })
				require.NoError(t, os.Symlink(extDir, filepath.Join(tmpDir, "link")))
				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Path: "link/secret.txt",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.ErrorContains(t, err, "attempts to traverse outside the working directory")
				// The external file must survive.
				_, statErr := os.Stat(
					filepath.Join(filepath.Dir(workDir), "kargo-ext-literal", "secret.txt"))
				assert.NoError(t, statErr)
				// The symlink itself is also retained
				_, lstatErr := os.Lstat(filepath.Join(workDir, "link"))
				assert.NoError(t, lstatErr)
			},
		},
		{
			// ".." cannot be expressed through os.DirFS (it refuses to traverse
			// above the root), so the pattern simply matches nothing
			name: "glob pattern with .. matches nothing",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				escapePath := filepath.Join(filepath.Dir(tmpDir), "escape-target")
				require.NoError(t, os.WriteFile(escapePath, []byte("x"), 0o600))
				t.Cleanup(func() { _ = os.Remove(escapePath) })
				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Path:          "../*",
				PathsAreGlobs: true,
				Strict:        true,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.ErrorContains(t, err, "no paths matched pattern")
				// Nothing outside workDir was deleted.
				_, statErr := os.Stat(filepath.Join(filepath.Dir(workDir), "escape-target"))
				assert.NoError(t, statErr)
			},
		},
		{
			name: "glob does not delete external files through a symlink",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()
				extDir := filepath.Join(filepath.Dir(tmpDir), "kargo-ext-target")
				require.NoError(t, os.Mkdir(extDir, 0o755))
				require.NoError(t, os.WriteFile(
					filepath.Join(extDir, "secret.txt"), []byte("x"), 0o600))
				t.Cleanup(func() { _ = os.RemoveAll(extDir) })
				require.NoError(t, os.Symlink(extDir, filepath.Join(tmpDir, "link")))
				return tmpDir
			},
			cfg: builtin.DeleteConfig{
				Path:          "link/*",
				PathsAreGlobs: true,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.ErrorContains(t, err, "attempts to traverse outside the working directory")
				// The external file must survive.
				_, statErr := os.Stat(
					filepath.Join(filepath.Dir(workDir), "kargo-ext-target", "secret.txt"))
				assert.NoError(t, statErr)
				// The symlink itself is also retained
				_, lstatErr := os.Lstat(filepath.Join(workDir, "link"))
				assert.NoError(t, lstatErr)
			},
		},
	}
	runner := &fileDeleter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := tt.setupFiles(t)
			result, err := runner.run(
				t.Context(),
				&promotion.StepContext{WorkDir: workDir},
				tt.cfg,
			)
			tt.assertions(t, workDir, result, err)
		})
	}
}
