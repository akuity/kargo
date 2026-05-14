package builtin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_fileWriter_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name: "path not specified",
			config: promotion.Config{
				"contents": "test content",
			},
			expectedProblems: []string{
				"invalid file-write config: (root): path is required",
			},
		},
		{
			name: "path is empty string",
			config: promotion.Config{
				"path":     "",
				"contents": "test content",
			},
			expectedProblems: []string{
				"invalid file-write config: path: String length must be greater than or equal to 1",
			},
		},
		{
			name: "contents not specified",
			config: promotion.Config{
				"path": "test.txt",
			},
			expectedProblems: []string{
				"invalid file-write config: (root): contents is required",
			},
		},
		{
			name: "contents may be empty",
			config: promotion.Config{
				"path":     "test.txt",
				"contents": "",
			},
		},
		{
			name: "unknown field",
			config: promotion.Config{
				"path":     "test.txt",
				"contents": "test content",
				"unknown":  true,
			},
			expectedProblems: []string{
				"invalid file-write config: (root): Additional property unknown is not allowed",
			},
		},
		{
			name: "valid configuration",
			config: promotion.Config{
				"path":      "test.txt",
				"contents":  "test content",
				"overwrite": true,
			},
		},
	}

	r := newFileWriter(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*fileWriter)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_fileWriter_run(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		dirs       []string
		cfg        builtin.FileWriteConfig
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "writes new file",
			cfg: builtin.FileWriteConfig{
				Path:     "out/config.yaml",
				Contents: "key: value\n",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				content, readErr := os.ReadFile(filepath.Join(workDir, "out/config.yaml"))
				require.NoError(t, readErr)
				assert.Equal(t, "key: value\n", string(content))
			},
		},
		{
			name: "writes empty file",
			cfg: builtin.FileWriteConfig{
				Path:     "empty.txt",
				Contents: "",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				content, readErr := os.ReadFile(filepath.Join(workDir, "empty.txt"))
				require.NoError(t, readErr)
				assert.Empty(t, content)
			},
		},
		{
			name: "fails when file exists without overwrite",
			files: map[string]string{
				"config.yaml": "existing: true\n",
			},
			cfg: builtin.FileWriteConfig{
				Path:     "config.yaml",
				Contents: "existing: false\n",
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.ErrorContains(t, err, "already exists")
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)

				content, readErr := os.ReadFile(filepath.Join(workDir, "config.yaml"))
				require.NoError(t, readErr)
				assert.Equal(t, "existing: true\n", string(content))
			},
		},
		{
			name: "overwrites existing file",
			files: map[string]string{
				"config.yaml": "existing: true\n",
			},
			cfg: builtin.FileWriteConfig{
				Path:      "config.yaml",
				Contents:  "existing: false\n",
				Overwrite: true,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusSucceeded}, result)

				content, readErr := os.ReadFile(filepath.Join(workDir, "config.yaml"))
				require.NoError(t, readErr)
				assert.Equal(t, "existing: false\n", string(content))
			},
		},
		{
			name: "fails when path escapes work dir",
			cfg: builtin.FileWriteConfig{
				Path:     "../escape.txt",
				Contents: "test content",
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				require.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
			},
		},
		{
			name: "fails when destination is directory",
			dirs: []string{"config.yaml"},
			cfg: builtin.FileWriteConfig{
				Path:      "config.yaml",
				Contents:  "test content",
				Overwrite: true,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				require.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
			},
		},
	}

	runner := &fileWriter{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workDir := t.TempDir()
			for _, dir := range tt.dirs {
				require.NoError(t, os.MkdirAll(filepath.Join(workDir, dir), 0o700))
			}
			for path, content := range tt.files {
				absPath := filepath.Join(workDir, path)
				require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0o700))
				require.NoError(t, os.WriteFile(absPath, []byte(content), 0o600))
			}

			result, err := runner.run(
				t.Context(),
				&promotion.StepContext{WorkDir: workDir},
				tt.cfg,
			)
			tt.assertions(t, workDir, result, err)
		})
	}
}
