package directives

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func Test_fileDeleter_runPromotionStep(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(*testing.T) string
		cfg        DeleteConfig
		assertions func(*testing.T, string, PromotionStepResult, error)
	}{
		{
			name: "succeeds deleting file",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				path := filepath.Join(tmpDir, "input.txt")
				require.NoError(t, os.WriteFile(path, []byte("test content"), 0o600))

				return tmpDir
			},
			cfg: DeleteConfig{
				Path: "input.txt",
			},
			assertions: func(t *testing.T, _ string, result PromotionStepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

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
			cfg: DeleteConfig{
				Path: "dirToDelete",
			},
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

				_, statErr := os.Stat(filepath.Join(workDir, "dirToDelete"))
				assert.True(t, os.IsNotExist(statErr))
			},
		},
		{
			name: "fails for non-existent path",
			setupFiles: func(t *testing.T) string {
				return t.TempDir()
			},
			cfg: DeleteConfig{
				Path: "nonExistentFile.txt",
			},
			assertions: func(t *testing.T, _ string, result PromotionStepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseErrored}, result)
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
			cfg: DeleteConfig{
				Path: "input/symlink.txt",
			},
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				assert.NoError(t, err)
				require.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

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
			cfg: DeleteConfig{
				Path: "foo/",
			},
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				assert.NoError(t, err)
				require.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

				_, statErr := os.Stat(filepath.Join(workDir, "foo", "file.txt"))
				assert.Error(t, statErr)
				assert.True(t, os.IsNotExist(statErr))

				_, statErr = os.Stat(filepath.Join(workDir, "bar", "file.txt"))
				assert.NoError(t, statErr)
			},
		},
	}
	runner := &fileDeleter{}

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
