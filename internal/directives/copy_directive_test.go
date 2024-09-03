package directives

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_copyDirective_run(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(*testing.T) string
		cfg        CopyConfig
		assertions func(*testing.T, string, Result, error)
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
			assertions: func(t *testing.T, workDir string, result Result, err error) {
				assert.Equal(t, ResultSuccess, result)
				assert.NoError(t, err)

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
			assertions: func(t *testing.T, workDir string, result Result, err error) {
				assert.Equal(t, ResultSuccess, result)
				assert.NoError(t, err)

				outDir := filepath.Join(workDir, "output")

				outPath := filepath.Join(outDir, "input.txt")
				b, err := os.ReadFile(outPath)
				assert.NoError(t, err)
				assert.Equal(t, "test content", string(b))

				nestedDir := filepath.Join(outDir, "nested")
				nestedPath := filepath.Join(nestedDir, "nested.txt")
				b, err = os.ReadFile(nestedPath)
				assert.NoError(t, err)
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
			assertions: func(t *testing.T, workDir string, result Result, err error) {
				assert.Equal(t, ResultSuccess, result)
				assert.NoError(t, err)

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
			name: "fails with invalid input path",
			setupFiles: func(t *testing.T) string {
				return t.TempDir()
			},
			cfg: CopyConfig{
				InPath: "input.txt",
			},
			assertions: func(t *testing.T, _ string, result Result, err error) {
				require.ErrorContains(t, err, "failed to copy")
				assert.Equal(t, ResultFailure, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := tt.setupFiles(t)

			d := &copyDirective{}
			result, err := d.run(context.Background(), &StepContext{WorkDir: workDir}, tt.cfg)
			tt.assertions(t, workDir, result, err)
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
