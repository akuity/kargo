package directives

import (
	"context"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
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
			assertions: func(t *testing.T, workDir string, result PromotionStepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, PromotionStepResult{Status: kargoapi.PromotionPhaseSucceeded}, result)

				_, statError := os.Stat("input.txt")
				assert.ErrorIs(t, statError, os.ErrNotExist)
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
				assert.ErrorIs(t, statErr, os.ErrNotExist)
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
