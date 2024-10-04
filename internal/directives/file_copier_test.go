package directives

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_fileCopier_runPromotionStep(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(*testing.T) string
		cfg        CopyConfig
		assertions func(*testing.T, string, PromotionStepResult, error)
	}{
		{
			name: "succeeds copying file",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				inPath := filepath.Join(tmpDir, "input.txt")
				require.NoError(t, os.WriteFile(inPath, []byte("test content"), 0o600))

				return tmpDir
			},
			cfg: CopyConfig{
				InPath:  "input.txt",
				OutPath: "output.txt",
			},
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

				outPath := filepath.Join(workDir, "output.txt")
				b, err := os.ReadFile(outPath)
				assert.NoError(t, err)
				assert.Equal(t, "test content", string(b))
			},
		},
		{
			name: "succeeds copying directory",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				inDir := filepath.Join(tmpDir, "input")
				require.NoError(t, os.Mkdir(inDir, 0o755))

				filePath := filepath.Join(inDir, "input.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("test content"), 0o600))

				nestedDir := filepath.Join(inDir, "nested")
				require.NoError(t, os.Mkdir(nestedDir, 0o755))
				nestedPath := filepath.Join(nestedDir, "nested.txt")
				require.NoError(t, os.WriteFile(nestedPath, []byte("nested content"), 0o600))

				return tmpDir
			},
			cfg: CopyConfig{
				InPath:  "input/",
				OutPath: "output/",
			},
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

				outDir := filepath.Join(workDir, "output")

				outPath := filepath.Join(outDir, "input.txt")
				b, err := os.ReadFile(outPath)
				require.NoError(t, err)
				assert.Equal(t, "test content", string(b))

				nestedDir := filepath.Join(outDir, "nested")
				nestedPath := filepath.Join(nestedDir, "nested.txt")
				b, err = os.ReadFile(nestedPath)
				require.NoError(t, err)
				assert.Equal(t, "nested content", string(b))
			},
		},
		{
			name: "ignores symlink",
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
			cfg: CopyConfig{
				InPath:  "input/",
				OutPath: "output/",
			},
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				assert.NoError(t, err)
				require.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

				outDir := filepath.Join(workDir, "output")

				outPath := filepath.Join(outDir, "input.txt")
				b, err := os.ReadFile(outPath)
				assert.NoError(t, err)
				assert.Equal(t, "test content", string(b))

				symlinkPath := filepath.Join(outDir, "symlink.txt")
				_, err = os.Lstat(symlinkPath)
				assert.Error(t, err)
				assert.True(t, os.IsNotExist(err))
			},
		},
		{
			name: "default ignore rule ignores .git directory",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				inDir := filepath.Join(tmpDir, "input")
				require.NoError(t, os.Mkdir(inDir, 0o755))

				ignoreFilePath := filepath.Join(inDir, ".git", "file.txt")
				require.NoError(t, os.MkdirAll(filepath.Dir(ignoreFilePath), 0o755))
				require.NoError(t, os.WriteFile(ignoreFilePath, []byte("ignored content"), 0o600))

				filePath := filepath.Join(inDir, "input.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("test content"), 0o600))

				return tmpDir
			},
			cfg: CopyConfig{
				InPath:  "input/",
				OutPath: "output/",
			},
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				assert.NoError(t, err)
				require.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

				outDir := filepath.Join(workDir, "output")

				outPath := filepath.Join(outDir, "input.txt")
				b, err := os.ReadFile(outPath)
				require.NoError(t, err)
				assert.Equal(t, "test content", string(b))

				ignoreFilePath := filepath.Join(outDir, ".git", "file.txt")
				require.NoDirExists(t, filepath.Dir(ignoreFilePath))
				require.NoFileExists(t, ignoreFilePath)
			},
		},
		{
			name: "default ignore rule ignores .git file",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				inDir := filepath.Join(tmpDir, "input")
				require.NoError(t, os.Mkdir(inDir, 0o755))

				ignoreFilePath := filepath.Join(inDir, ".git")
				require.NoError(t, os.WriteFile(ignoreFilePath, []byte("ignored content"), 0o600))

				filePath := filepath.Join(inDir, "input.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("test content"), 0o600))

				return tmpDir
			},
			cfg: CopyConfig{
				InPath:  "input/",
				OutPath: "output/",
			},
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				assert.NoError(t, err)
				require.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

				outDir := filepath.Join(workDir, "output")

				outPath := filepath.Join(outDir, "input.txt")
				b, err := os.ReadFile(outPath)
				require.NoError(t, err)
				assert.Equal(t, "test content", string(b))

				ignoreFilePath := filepath.Join(outDir, ".git")
				require.NoFileExists(t, ignoreFilePath)
			},
		},
		{
			name: "overwrite default ignore rules",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				inDir := filepath.Join(tmpDir, "input")
				require.NoError(t, os.Mkdir(inDir, 0o755))

				ignoreFilePath := filepath.Join(inDir, ".git", "file.txt")
				require.NoError(t, os.MkdirAll(filepath.Dir(ignoreFilePath), 0o755))
				require.NoError(t, os.WriteFile(ignoreFilePath, []byte("included content"), 0o600))

				filePath := filepath.Join(inDir, "input.txt")
				require.NoError(t, os.WriteFile(filePath, []byte("test content"), 0o600))

				return tmpDir
			},
			cfg: CopyConfig{
				InPath:  "input/",
				OutPath: "output/",
				Ignore:  "!.git/",
			},
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				assert.NoError(t, err)
				require.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

				outDir := filepath.Join(workDir, "output")

				outPath := filepath.Join(outDir, "input.txt")
				b, err := os.ReadFile(outPath)
				require.NoError(t, err)
				assert.Equal(t, "test content", string(b))

				ignoreFilePath := filepath.Join(outDir, ".git", "file.txt")
				b, err = os.ReadFile(ignoreFilePath)
				require.NoError(t, err)
				assert.Equal(t, "included content", string(b))
			},
		},
		{
			name: "fails with invalid input path",
			setupFiles: func(t *testing.T) string {
				return t.TempDir()
			},
			cfg: CopyConfig{
				InPath: "input.txt",
			},
			assertions: func(t *testing.T, _ string, result PromotionStepResult, err error) {
				require.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, result)
				require.ErrorContains(t, err, "failed to copy")
			},
		},
	}

	runner := &fileCopier{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := tt.setupFiles(t)
			result, err := runner.runPromotionStep(
				context.Background(),
				&PromotionStepContext{WorkDir: workDir},
				tt.cfg,
			)
			tt.assertions(t, workDir, result, err)
		})
	}
}

func Test_fileCopier_loadIgnoreRules(t *testing.T) {
	tests := []struct {
		name       string
		inPath     string
		rules      string
		setup      func(*testing.T) string
		assertions func(*testing.T, string, gitignore.Matcher, error)
	}{
		{
			name:   "directory path",
			inPath: "testdir",
			rules: `*.txt
# comment
*.go`,
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			assertions: func(t *testing.T, inPath string, matcher gitignore.Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				basePath := strings.Split(inPath, string(filepath.Separator))

				// Provided rules
				assert.True(t, matcher.Match(append(basePath, "file.txt"), false))
				assert.True(t, matcher.Match(append(basePath, "file.go"), false))
				assert.False(t, matcher.Match(append(basePath, "file.log"), false))

				// Default rules
				assert.True(t, matcher.Match(append(basePath, ".git"), true))
				assert.True(t, matcher.Match(append(basePath, ".git", "file.log"), true))
			},
		},
		{
			name:  "file path",
			rules: "*.log",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				inPath := filepath.Join(dir, "testfile.txt")
				assert.NoError(t, os.WriteFile(inPath, []byte("test"), 0o600))
				return inPath
			},
			assertions: func(t *testing.T, inPath string, matcher gitignore.Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				basePath := strings.Split(filepath.Dir(inPath), string(filepath.Separator))
				assert.True(t, matcher.Match(append(basePath, "something.log"), false))
				assert.False(t, matcher.Match(append(basePath, "testfile.txt"), false))
				assert.True(t, matcher.Match(append(basePath, ".git", "file"), false))
			},
		},
		{
			name:   "non-existent path",
			inPath: "nonexistent",
			rules:  "*.tmp",
			setup: func(*testing.T) string {
				return "does-not-exist"
			},
			assertions: func(t *testing.T, _ string, matcher gitignore.Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)
			},
		},
		{
			name:   "empty rules",
			inPath: "testdir",
			rules:  "",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			assertions: func(t *testing.T, inPath string, matcher gitignore.Matcher, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, matcher)

				basePath := strings.Split(inPath, string(filepath.Separator))
				assert.True(t, matcher.Match(append(basePath, ".git", "file"), false))
				assert.False(t, matcher.Match(append(basePath, "file.txt"), false))
			},
		},
		{
			name: "invalid path",
			setup: func(*testing.T) string {
				return string([]byte{0})
			},
			rules: "*.tmp",
			assertions: func(t *testing.T, _ string, matcher gitignore.Matcher, err error) {
				assert.ErrorContains(t, err, "failed to determine domain")
				assert.Nil(t, matcher)
			},
		},
	}

	for _, tt := range tests {
		f := &fileCopier{}

		t.Run(tt.name, func(t *testing.T) {
			inPath := tt.setup(t)
			matcher, err := f.loadIgnoreRules(inPath, tt.rules)
			tt.assertions(t, inPath, matcher, err)
		})
	}
}

func Test_sanitizePathError(t *testing.T) {
	tests := []struct {
		name       string
		workDir    string
		err        error
		assertions func(*testing.T, error)
	}{
		{
			name:    "PathError with relative path",
			workDir: "/tmp/work/dir",
			err:     &os.PathError{Op: "open", Path: "/tmp/work/dir/file.txt", Err: os.ErrNotExist},
			assertions: func(t *testing.T, result error) {
				var pathErr *os.PathError
				assert.True(t, errors.As(result, &pathErr))
				assert.Equal(t, "open", pathErr.Op)
				assert.Equal(t, "file.txt", pathErr.Path)
				assert.Equal(t, os.ErrNotExist, pathErr.Err)
			},
		},
		{
			name:    "PathError with path outside workDir",
			workDir: "/tmp/work/dir",
			err:     &os.PathError{Op: "read", Path: "/etc/config.ini", Err: os.ErrPermission},
			assertions: func(t *testing.T, result error) {
				var pathErr *os.PathError
				assert.True(t, errors.As(result, &pathErr))
				assert.Equal(t, "read", pathErr.Op)
				assert.Equal(t, "config.ini", pathErr.Path)
				assert.Equal(t, os.ErrPermission, pathErr.Err)
			},
		},
		{
			name:    "non-PathError",
			workDir: "/tmp/work/dir",
			err:     errors.New("generic error"),
			assertions: func(t *testing.T, result error) {
				assert.Equal(t, "generic error", result.Error())
			},
		},
		{
			name:    "PathError with workDir",
			workDir: "/tmp/work/dir",
			err:     &os.PathError{Op: "stat", Path: "/tmp/work/dir", Err: os.ErrNotExist},
			assertions: func(t *testing.T, result error) {
				var pathErr *os.PathError
				errors.As(result, &pathErr)
				assert.Equal(t, "stat", pathErr.Op)
				assert.Equal(t, ".", pathErr.Path)
				assert.Equal(t, os.ErrNotExist, pathErr.Err)
			},
		},
		{
			name: "nil error",
			err:  nil,
			assertions: func(t *testing.T, result error) {
				assert.Nil(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePathError(tt.err, tt.workDir)
			tt.assertions(t, result)
		})
	}
}
