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

func Test_yamlMerger_validate(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]any
		expectedError string
	}{
		{
			name: "inPaths not specified (missing path field)",
			config: map[string]any{
				"outPath": "valid.yaml",
			},
			expectedError: "(root): inPaths is required",
		},
		{
			name: "inPaths field is an empty array",
			config: map[string]any{
				"inPaths": []string{},
				"outPath": "valid.yaml",
			},
			expectedError: "invalid yaml-merge config: inPaths: Array must have at least 1 items",
		},
		{
			name: "inPaths contains empty string",
			config: map[string]any{
				"inPaths": []string{""},
				"outPath": "valid.yaml",
			},
			expectedError: "",
		},
		{
			name: "outPath not specified (missing path field)",
			config: map[string]any{
				"inPaths": []string{"valid.yaml"},
			},
			expectedError: "invalid yaml-merge config: (root): outPath is required",
		},
		{
			name: "outPath is empty string",
			config: map[string]any{
				"inPaths": []string{"valid.yaml"},
				"outPath": "",
			},
			expectedError: "invalid yaml-merge config: outPath: String length must be greater than or equal to 1",
		},
		{
			name: "valid configuration (inPaths + outPath present)",
			config: map[string]any{
				"inPaths": []string{"valid.yaml"},
				"outPath": "valid.yaml",
			},
			expectedError: "",
		},
	}

	r := newYAMLMerger()
	runner, ok := r.(*yamlMerger)
	require.True(t, ok)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := runner.validate(tc.config)
			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}

func Test_YAMLMerger_run(t *testing.T) {
	tests := []struct {
		name       string
		stepCtx    *promotion.StepContext
		cfg        builtin.YAMLMergeConfig
		files      map[string]string
		assertions func(*testing.T, string, promotion.StepResult, error)
	}{
		{
			name: "successful run with modified outputs",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.YAMLMergeConfig{
				InPaths: []string{"base.yaml", "overrides.yaml"},
				OutPath: "modified.yaml",
			},
			files: map[string]string{
				"base.yaml": `
app:
  version: "1.0.0"
features:
  newFeature: false
`,
				"overrides.yaml": `
app:
  version: "2.0.0"
`,
			},
			assertions: func(t *testing.T, workDir string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				content, err := os.ReadFile(path.Join(workDir, "modified.yaml"))
				require.NoError(t, err)
				assert.Contains(t, string(content), `  version: "2.0.0"`)
				assert.Contains(t, string(content), `  newFeature: false`)
			},
		},
		{
			name: "failed to read InPaths file in Strict mode",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.YAMLMergeConfig{
				InPaths: []string{"non-existent/values.yaml"},
				OutPath: "modified.yaml",
				Strict:  true,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{Status: kargoapi.PromotionStepStatusErrored}, result)
				assert.Contains(t, err.Error(), "error reading file")
			},
		},
		{
			name: "failed to read InPaths file",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.YAMLMergeConfig{
				InPaths: []string{"non-existent/values.yaml"},
				OutPath: "modified.yaml",
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.NoError(t, err)
				assert.Equal(t, promotion.StepResult{
					Status: kargoapi.PromotionStepStatusSucceeded,
					Output: map[string]any{
						"commitMessage": "Merged YAML files to modified.yaml\n",
					},
				}, result)
			},
		},
		{
			name: "no outputs provided",
			stepCtx: &promotion.StepContext{
				Project: "test-project",
			},
			cfg: builtin.YAMLMergeConfig{
				InPaths: []string{"base.yaml", "overrides.yaml"},
				OutPath: "",
			},
			files: map[string]string{
				"base.yaml": `
app:
  version: "1.0.0"
features:
  newFeature: false
`,
				"overrides.yaml": `
app:
  version: "2.0.0"
`,
			},
			assertions: func(t *testing.T, _ string, result promotion.StepResult, err error) {
				assert.Error(t, err)
				assert.Equal(t, promotion.StepResult{
					Status: kargoapi.PromotionStepStatusErrored,
				}, result)
				assert.Contains(t, err.Error(), "Error writing to file")
			},
		},
	}

	runner := &yamlMerger{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepCtx := tt.stepCtx

			stepCtx.WorkDir = t.TempDir()
			for p, c := range tt.files {
				require.NoError(t, os.MkdirAll(path.Join(stepCtx.WorkDir, path.Dir(p)), 0o700))
				require.NoError(t, os.WriteFile(path.Join(stepCtx.WorkDir, p), []byte(c), 0o600))
			}

			result, err := runner.run(context.Background(), stepCtx, tt.cfg)
			tt.assertions(t, stepCtx.WorkDir, result, err)
		})
	}
}
