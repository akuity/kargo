package builtin

import (
	"context"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
	"github.com/akuity/kargo/pkg/x/promotion/runner/builtin"
)

func Test_yamlMerger_convert(t *testing.T) {
	tests := []validationTestCase{
		{
			name: "inFiles not specified (missing path field)",
			config: map[string]any{
				"outFile": "valid.yaml",
			},
			expectedProblems: []string{
				"(root): inFiles is required",
			},
		},
		{
			name: "inFiles field is an empty array",
			config: map[string]any{
				"inFiles": []string{},
				"outFile": "valid.yaml",
			},
			expectedProblems: []string{
				"invalid yaml-merge config: inFiles: Array must have at least 1 items",
			},
		},
		{
			name: "inFiles contains empty string",
			config: map[string]any{
				"inFiles": []string{""},
				"outFile": "valid.yaml",
			},
			expectedProblems: nil,
		},
		{
			name: "outFile not specified (missing path field)",
			config: map[string]any{
				"inFiles": []string{"valid.yaml"},
			},
			expectedProblems: []string{
				"invalid yaml-merge config: (root): outFile is required",
			},
		},
		{
			name: "outFile is empty string",
			config: map[string]any{
				"inFiles": []string{"valid.yaml"},
				"outFile": "",
			},
			expectedProblems: []string{
				"invalid yaml-merge config: outFile: String length must be greater than or equal to 1",
			},
		},
		{
			name: "valid configuration (inFiles + outFile present)",
			config: map[string]any{
				"inFiles": []string{"valid.yaml"},
				"outFile": "valid.yaml",
			},
			expectedProblems: nil,
		},
	}

	r := newYAMLMerger(promotion.StepRunnerCapabilities{})
	runner, ok := r.(*yamlMerger)
	require.True(t, ok)

	runValidationTests(t, runner.convert, tests)
}

func Test_YAMLMerger_run(t *testing.T) {
	tests := []struct {
		name       string
		cfg        builtin.YAMLMergeConfig
		files      map[string]string
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "successful merge with multiple files",
			cfg: builtin.YAMLMergeConfig{
				InFiles: []string{"base.yaml", "overrides.yaml"},
				OutFile: "modified.yaml",
			},
			files: map[string]string{
				"base.yaml": `app:
  version: "1.0.0"
features:
  newFeature: false
`,
				"overrides.yaml": `app:
  version: "2.0.0"
`,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				require.NoError(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusSucceeded, result.Status)

				content, err := os.ReadFile(path.Join(workDir, "modified.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), `version: "2.0.0"`)
				assert.Contains(t, string(content), `newFeature: false`)

				assert.NotNil(t, result.Output)
				commitMsg, ok := result.Output["commitMessage"].(string)
				require.True(t, ok)
				assert.Contains(t, commitMsg, "Merged 2 YAML files")
			},
		},
		{
			name: "error when input file not found and ignoreMissingFiles is false",
			cfg: builtin.YAMLMergeConfig{
				InFiles:            []string{"non-existent.yaml"},
				OutFile:            "output.yaml",
				IgnoreMissingFiles: false,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				require.Error(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
				assert.Contains(t, err.Error(), `input file "non-existent.yaml" not found`)
			},
		},
		{
			name: "error when all files missing and ignoreMissingFiles is true",
			cfg: builtin.YAMLMergeConfig{
				InFiles:            []string{"missing1.yaml", "missing2.yaml"},
				OutFile:            "output.yaml",
				IgnoreMissingFiles: true,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				require.Error(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
				assert.Contains(t, err.Error(), "no input files found to merge")
			},
		},
		{
			name: "error when output path is invalid",
			cfg: builtin.YAMLMergeConfig{
				InFiles: []string{"base.yaml"},
				OutFile: "",
			},
			files: map[string]string{
				"base.yaml": `app:
  version: "1.0.0"
`,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				require.Error(t, err)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
				assert.Contains(t, err.Error(), "error merging YAML files")
			},
		},
	}

	runner := &yamlMerger{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepCtx := &promotion.StepContext{
				Project: "test-project",
				WorkDir: t.TempDir(),
			}

			// Setup test files
			for filePath, content := range tt.files {
				fullPath := path.Join(stepCtx.WorkDir, filePath)
				require.NoError(t, os.MkdirAll(path.Dir(fullPath), 0o700))
				require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o600))
			}

			result, err := runner.run(context.Background(), stepCtx, tt.cfg)
			tt.assertions(t, stepCtx.WorkDir, result, err)
		})
	}
}

func Test_YAMLMerger_generateCommitMessage(t *testing.T) {
	tests := []struct {
		name     string
		outPath  string
		inFiles  []string
		expected string
	}{
		{
			name:    "multiple files",
			outPath: "out/test.yaml",
			inFiles: []string{"base.yaml", "overrides.yaml"},
			expected: `Merged 2 YAML files to out/test.yaml
- base.yaml
- overrides.yaml`,
		},
		{
			name:     "single file",
			outPath:  "out/test.yaml",
			inFiles:  []string{"base.yaml"},
			expected: "Merged base.yaml to out/test.yaml",
		},
		{
			name:     "no files returns empty string",
			outPath:  "out/test.yaml",
			inFiles:  []string{},
			expected: "",
		},
	}

	runner := &yamlMerger{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runner.generateCommitMessage(tt.outPath, tt.inFiles)
			assert.Equal(t, tt.expected, result)
		})
	}
}
