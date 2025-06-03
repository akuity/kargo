package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_fileWriter_run(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(*testing.T) string
		cfg        builtin.WriteConfig
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "succeeds writing file",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				return tmpDir
			},
			cfg: builtin.WriteConfig{
				Contents: "test content",
				OutFile:  "output.txt",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				outPath := filepath.Join(workDir, "output.txt")
				b, err := os.ReadFile(outPath)
				assert.NoError(t, err)
				assert.Equal(t, "test content", string(b))
			},
		},
		{
			name: "succeeds writing file in directory",
			setupFiles: func(t *testing.T) string {
				tmpDir := t.TempDir()

				return tmpDir
			},
			cfg: builtin.WriteConfig{
				Contents: "test content",
				OutFile:  "/newdir/output.txt",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				outPath := filepath.Join(workDir, "/newdir/output.txt")
				b, err := os.ReadFile(outPath)
				assert.NoError(t, err)
				assert.Equal(t, "test content", string(b))
			},
		},
		{
			name: "fails with invalid output file",
			setupFiles: func(t *testing.T) string {
				return t.TempDir()
			},
			cfg: builtin.WriteConfig{
				Contents: "test content",
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				require.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				require.ErrorContains(t, err, "failed to write")
			},
		},
		{
			name: "fails with directory collision",
			setupFiles: func(t *testing.T) string {

				tempDir := t.TempDir()
				assert.NoError(t, os.MkdirAll(filepath.Join(tempDir, "existing_dir/output"), 0755))
				return tempDir
			},
			cfg: builtin.WriteConfig{
				Contents: "test content",
				OutFile:  "/existing_dir/output",
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				require.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				require.ErrorContains(t, err, "failed to write")
			},
		},
	}

	runner := &fileWriter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := tt.setupFiles(t)
			result, err := runner.run(
				context.Background(),
				&promotion.StepContext{WorkDir: workDir},
				tt.cfg,
			)
			tt.assertions(t, workDir, result, err)
		})
	}
}
