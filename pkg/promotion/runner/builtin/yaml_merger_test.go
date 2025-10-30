package builtin

import (
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
	testCases := []validationTestCase{
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

	runValidationTests(t, runner.convert, testCases)
}

func Test_YAMLMerger_run(t *testing.T) {
	testCases := []struct {
		name       string
		fsFiles    map[string]string
		cfg        builtin.YAMLMergeConfig
		assertions func(*testing.T, promotion.StepResult, error)
	}{
		{
			name: "input file not found with ignoreMissingFiles false",
			cfg: builtin.YAMLMergeConfig{
				InFiles:            []string{"non-existent.yaml"},
				IgnoreMissingFiles: false,
			},
			assertions: func(t *testing.T, result promotion.StepResult, err error) {
				assert.Contains(t, err.Error(), `input file "non-existent.yaml" not found`)
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
			},
		},
		{
			name: "error merging files",
			fsFiles: map[string]string{
				"invalid.yaml": ":",
			},
			cfg: builtin.YAMLMergeConfig{
				InFiles: []string{"invalid.yaml"},
				OutFile: "merged.yaml",
			},
			assertions: func(t *testing.T, result promotion.StepResult, err error) {
				assert.Contains(t, err.Error(), "error merging YAML files")
				assert.Equal(t, kargoapi.PromotionStepStatusErrored, result.Status)
			},
		},
		{
			name: "successful merge",
			fsFiles: map[string]string{
				"base.yaml":    "version: 1.0.0",
				"overlay.yaml": "version: 2.0.0",
			},
			cfg: builtin.YAMLMergeConfig{
				InFiles: []string{
					"base.yaml",
					"overlay.yaml",
					"non-existent.yaml", // We should get past this
				},
				OutFile:            "merged.yaml",
				IgnoreMissingFiles: true,
			},
			assertions: func(t *testing.T, result promotion.StepResult, err error) {
				require.NoError(t, err)
				// Note: We're not testing the results of the merge itself because
				// yaml.MergeFiles is well-tested.
				commitMsg, ok := result.Output["commitMessage"].(string)
				require.True(t, ok)
				assert.Contains(t, commitMsg, "Merged 2 YAML files")
			},
		},
	}

	runner := &yamlMerger{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			workDir := t.TempDir()
			for filePath, content := range testCase.fsFiles {
				err := os.WriteFile(
					path.Join(workDir, filePath),
					[]byte(content),
					0o600,
				)
				require.NoError(t, err)
			}
			stepCtx := &promotion.StepContext{WorkDir: workDir}
			result, err := runner.run(t.Context(), stepCtx, testCase.cfg)
			testCase.assertions(t, result, err)
		})
	}
}

func Test_YAMLMerger_generateCommitMessage(t *testing.T) {
	testCases := []struct {
		name     string
		outPath  string
		inFiles  []string
		expected string
	}{
		{
			name:     "no input files",
			outPath:  "out/test.yaml",
			inFiles:  []string{},
			expected: "",
		},
		{
			name:     "one input file",
			outPath:  "out/test.yaml",
			inFiles:  []string{"base.yaml"},
			expected: "Wrote base.yaml to out/test.yaml",
		},
		{
			name:    "multiple input files",
			outPath: "out/test.yaml",
			inFiles: []string{
				"base.yaml",
				"overrides.yaml",
			},
			expected: `Merged 2 YAML files to out/test.yaml
- base.yaml
- overrides.yaml`,
		},
	}

	runner := &yamlMerger{}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := runner.generateCommitMessage(testCase.inFiles, testCase.outPath)
			assert.Equal(t, testCase.expected, result)
		})
	}
}
